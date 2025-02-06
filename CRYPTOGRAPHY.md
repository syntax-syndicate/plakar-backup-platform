# cryptography design documentation

## Notes

- Whenever it is written that data was randomly-generated, it is using the `crypto/rand` PRNG as its source of randomness.
- The passphrase that protects a repository is the single most important secret:
if it leaks, content is no longer secret; if it is lost, content is no longer restorable.
- Repository can be stored in a public cloud, configuration and encrypted content being available to the hosting company and its employees
- Hashing algorithm, encryption algorithm and KDF were made configurable: we don't expose the choices yet and hardcode configuration, but software was written assuming they could be swapped and immutable repository configuration defines the choices in place.

**The encryption and decryption function are described at the end of this document. It can be assumed that anything described as being encrypted has gone through the same function taking the master key and a cleartext input to produce the encrypted output with no additional operations.**


## Initialization of an encrypted repository

User creating the repository is prompted for a passphrase that is passed through an entropy-based strength check to refuse weak ones:

```go
https://github.com/PlakarKorp/plakar/pull/413/files#diff-27daab5a56d87beab7cb5ede5778b561c3e712c9a6c8b56a2ddda3eb0d88dc00R110-R119
```

A 256-bits master key for the repository is derived from the passphrase using a KDF.
The code is ready to allow swapping the KDF should we want to change it.
So far we have used `scrypt` with a 128-bits randomly-generated salt and parameters N=32768, r=8, p=1 as suggested by the official package documentation but are considering switching to `argon2`.

- Q1: Should we switch to Argon2 or remain on Scrypt ?
- Q2: If we remain on Scrypt, are these parameters fine as of 2025 ?
- Q3: If we switch to Argon2, do you have recommendations ?
- Q4: Do you recommend switching to something else ?

In addition,
a 32-bytes block is also randomly generated and `AES256-GCM` encrypted with master key using the method described at the end of this document.

The KDF parameters, salt and encrypted block are then stored in the repository configuration file which remains in cleartext as it needs to let clients determine their setup to work with that repository. The following informations are therefore cleartext:

```go
type Configuration struct {
	Algorithm string    // "AES256-GCM"
	KDF       string    // "SCRYPT"
	KDFParams KDFParams // KDF-dependant parameters (see below)
	Canary    []byte    // random block for client passphrase validation
}

// for SCRYPT
type KDFParams struct {
	Salt   [saltSize]byte
	N      int
	R      int
	P      int
	KeyLen int
}

// for ARGON2 would be:
type KDFParams struct {
	Salt            [saltSize]byte
	Memory          int
	Iterations      int
	Parallelism     int
	KeyLen          int
}

```

## Opening of an encrypted repository

- The client retrieves the repository configuration and obtains the KDF parameters, the salt as well as the encrypted random 32-bytes block.
- It prompts user for the passphrase and derives the master key using the KDF parameters and salt.
- It verifies that the derived master key successfully decrypts the canary block using the `AES256-GCM` integrity check.
- If decryption works, master key is used during the session, otherwise client errors out due to repository passphrase mismatch.


## Internal structure of a repository

A repository consists of three main elements:

- an immutable configuration created during repository initialization
- packfiles
- state files

### Configuration
The configuration supposedly contains no sensitive informations.
It contains a UUID, a timestamp, the chunking settings for deduplication, the settings for compression (algorithm, level, ...) and the settings for encryption (hashing algorithm, encryption algorithm, key derivation algorithm and parameters as seen above).

### Packfiles
Packfiles are immutable structured files that are created during the backup process and which consist of three sections:

```
[DATA]
    [compressed/encrypted blob #1]
    [compressed/encrypted blob #2]
    [compressed/encrypted blob #3]
    [...]

[ENCRYPTED INDEX]
    [blob #1 type] [HMAC(checksum(data #1), master key)] [offset] [length]
    [blob #2 type] [HMAC(checksum(data #2), master key)] [offset] [length]
    [blob #3 type] [HMAC(checksum(data #3), master key)] [offset] [length]
    [...]

[ENCRYPTED FOOTER]
    Format version, creation timestamp, index offset and length
```

#### Data section
The data section contains a sequence of blobs that were created during the backup process by individually compressing and encrypting chunks of data of varying types and sizes.
These blobs are written one after the other in the packfile with no delimitation marker so that it's not possible to determine where they begin and where they end: the data section is just a stream of apparently random bytes.

Since blobs are created from a variety of data structures that are not related to file content but also to filesystem structure and software internal structures,
and because the backup process parallelizes packfiles filling, filesystem discovery and file content chunking,
two contiguous blobs in a packfile almost never represent data that is contiguous in the data source being backed up.
A file content has blobs spread across multiple packfiles and even within a same packfile it is almost always interlaced with unrelated blobs that do not necessarily originate from file content.


#### Index section
The index section contains a sequence of fixed-length records that contain the blob type, a HMAC of the original data checksum using the repository master key as secret, the offset of the blob in the data section and its length. The HMAC is used with the original data checksum as input so that application can use regular checksums to compare backups with a live filesystem but so that when lookups happen in the repository, these checksums are obfuscated and converted to a stable hmac transparently. A user may compute the checksum of /etc/passwd, expect that checksum to appear in the backup, but lookups transparently convert the checksum to the hmac before performing the lookup in a packfile.

The packfile index is compressed and encrypted as a whole. it is rarely used as the repository state files provide a unified index of all packfiles indexes, and is there as replica to help in the event of a repository state corruption. If needed, we can locate its position and size thanks to the footer, and we can decrypt it to access content.


#### Footer section
This fixed-size structure allows us to immediately locate the offset of the index and its size.
It does not contain sensitve data but is encrypted so as not to leak the index offset and make it harder to determine the number of blobs in a packfile: a packfile of ~20MB may contain 2x ~10MB blobs or 20x ~1MB blobs, only the index can help determine this.


### State files
State files are an event log of changes made to the repository to track the addition and deletion of data.
Every time a backup is done,
a new state file is created that contains only records for new blobs and packfiles.
If a snapshot is deleted, a new delta is created with a record to indicate the deletion.

These deltase consists of a serie of records to track some metadata, blob locations, deletion events and known packfiles identifiers.
A sample could look like this:

```
[...]
[ET_LOCATIONS] [RECORD SIZE] [HMAC(checksum(data #1), master key)] [packfile #3 identifier] [offset] [length]
[ET_LOCATIONS] [RECORD SIZE] [HMAC(checksum(data #2), master key)] [packfile #1 identifier] [offset] [length]
[ET_LOCATIONS] [RECORD SIZE] [HMAC(checksum(data #3), master key)] [packfile #2 identifier] [offset] [length]
[ET_PACKFILE] [RECORD SIZE] [packfile #1 identifier]
[ET_PACKFILE] [RECORD SIZE] [packfile #2 identifier]
[ET_PACKFILE] [RECORD SIZE] [packfile #3 identifier]
[...]
```

State files are compressed and encrypted as whole, they are not meant to be read and consumed directly.


## State synchronization

Everytime a client opens a repository, regardless of the operation being performed, a state synchronization is performed.

The client requests the repository for a list of state identifiers and fetches the states that are missing from its local cache.
State are read through a stream decoder that performs decryption and decompression on the flight,
and the local state cache is updated record by record.

At the end of the synchronization,
the local state cache has a full representation of the repository state that allows mapping blobs to packfile locations without querying the repository:
an existence check or the lookup of a blob location (packfile, offset length) does not result in a repository hit.


## Backup process

A snapshot header is created with a random 256-bits identifier and an empty snapshot state is created to hold changes to the current local state.

As a backup is performed, chunks of data are produced from various sources:
file content, filesystem structure, plakar internal structures, ... and a checksum of the data is computed for internal purposes and recording within the snapshot itself.

```
dataChecksum = sha256(data)
```

The backup process then needs to determine if these chunks exist in the repository already or if they need to be pushed.
To do so it computes a blob identifier by performing an hmac of the checksum with the master key:

```
blobId = hmac-sha256(dataChecksum, masterKey)
```
and uses that blob identifier as a key to query the local state.

```
found = localState.Lookup(blobId)
```

If the blob is found, then an object with the same checksum exists and there is no need to push it again.
If the blob is not found,
then the data is compressed and encrypted and passed to a packing job in charge of crafting new packfiles (always several in parallel).
The packing job will pick a random packfile, append the encrypted blob to its data section and update its index with the blob identifier:

```
[blobType] [blobId] [offset] [size]
```

When the packfile reaches a certain size (~20MB as of today, might be bumped), its index and footer are encrypted individually.
The hmac of the packfile is computed and the packfile is pushed to the repository with its hmac as packfile identifier:

```
packfileId = hmac-sha256(packfile, master key)
```

The snapshot state is then updated to indicate the creation of a packfile identified by `packfileId` and to indicate the mapping of the blob identified by `blobId` to that packfile along with the offset in the packfile data section and the blob size.

The snapshot header contains some of the checksums that are needed to rebuild a full view of the backup.
It is pushed as an encrypted blob to the packer, going through the same process as described above.

Finally,
the snapshot state is compressed and encrypted,
and pushed to the repository leading to an effective commit that makes the snapshot visible to other clients as they synchronize their states.


## Restore process

In a very simplified view,
a snapshot is a virtual filesystem where data is structured logically as directories -> files -> data.

As such,
a snapshot can be restored as whole,
just as it can be restored partially by providing a directory or file from where the restoration starts.

The snapshot header contains, among other things, the blob identifier for the root of the tree.
Restoration of a resource consists in fetching the root blob and decrypting it,
then progressively fetching blob for each children node leading to the data to be restored.
Each of the directory nodes contain filesystem informations,
but the file nodes also contain the identifier for a blob that describes the structure of the file content.
That structure contains the list of data blobs necessary to rebuild the file,
so restoring can be done by fetching, decrypting and writing the content to the target location on the restore filesystem.

As the snapshot header and each node is encrypted, the restore process is essentially a loop of lookups and decryptions,
some leading to mapping structures in memory and others to restoring data to a file.


## Check process

During backup,
when chunking files,
the checksum of each individual chunk and the checksum of the complete data is computed and stored as part of blobs.

It is possible to perform snapshot verifications of two kinds:

### Fast check

In a fast check, the client identifies the blobs necessary to rebuild the portion of the backup being checked.
It then looks up its local state to verify that each of these blobs are mapped to a packfile,
indicating that they are known to the repository.

As long as packfiles do not disappear from the repository leading to an incorrect mapping,
if all lookups succeed it implies that the backup would be recoverable.


### Slow check

In a slow check, the client identifies the blobs necessary to rebuild the portion of the backup being checked.
It looks up the location of these blobs in the local state and fetches them from the repository,
recomputing checksums to compare with recorded one.

This implies a full read of the backup from the repository but since checksums can be computed from sequential reads,
this is similar to a restoration in /dev/null (no writes to local disk) with the benefit of validating that repository has the data.


## Encryption and decryption functions

At the exception of the above, all encryption and decryption in `plakar` takes place through the same two functions.

Encryption is performed using `AES256-GCM` with individual 256-bits subkeys randomly-generated and protected by the master key, so that in the event of one subkey leaking or being broken for any reason unrelated to the master key leaking, then not all data would be immediately at risk.


### Encryption
The encryption works as follows:

    1- an encryption function takes a master key and an input buffer
    2- it randomly generates a 256-bits subkey and a subkey nonce
    3- it AES25-GCM encrypts the input buffer with that subkey into an output buffer
    4- a random master key nonce is generated
    5- subkey and subkey nonce are AES256-GCM encrypted with master key and master key nonce into a subkey block prepended to output buffer

    [encrypted subkey block] [encrypted data block]
     ^^^^^^^^^^^^^^^^^^^^^^   ^^^^^^^^^^^^^^^^^^^^
     master key encrypted     subkey encrypted


Q1: is it a safe scheme ?
Q2: how to generate a safe nounce for the subkey block as to ensure there is no reuse ?


### Decryption
The decryption works as follows:

    1- a decryption function takes a master key and an input buffer
    2- the input buffer is split into two parts: the subkey block and the data block
    3- the subkey block is decrypted so that subkey and subkey nonce are retrieved, GCM integrity check validates master key
    4- the data block is decrypted with the subkey and subkey nonce, GCM integrity check validates subkey
