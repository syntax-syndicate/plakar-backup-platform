# cryptography design documentation

## Notes

- Whenever it is written that data is randomly-generated, it is using the `crypto/rand` PRNG as its source of randomness.
- The passphrase that protects a repository is the single most important secret:
if it leaks, content is no longer secret; if it is lost, content is no longer restorable.
- Repositories may be stored in a public cloud, configuration and encrypted content being available to the hosting company and its employees.


## Current defaults

Hashing algorithm, encryption algorithm and KDF are all technically configurable even though we froze sane defaults and do not allow configuration yet.

- **MAC**: HMAC-BLAKE3
- **Data encryption**: AES256-GCM-SIV (with keySize=256bits)
- **Subkey encryption**: AES256-KW (with keySize=256bits)
- **KDF**: ARGON2ID (with time=4, Memory=256M, Threads=NumCPU, SaltSize=16, KeyLen=32)


## Terminology

- **Data** : raw data coming from structures or files
- **Chunks** : cleartext file content is split into chunks of data by the use of a Content-Defined Chunking algorithm
- **Blobs** : data that has been _encoded_ (compressed, encrypted, or compressed AND encrypted depending on configuration)


## Encryption and decryption

**Encryption and decryption function are described at the end of this document.**

**It can be assumed that anything encrypted has gone through the same function which takes the master key and a cleartext input to produce an encrypted output.**


## Encoding and decoding

In practice, the software never calls the encryption function directly.

Given an encryption key and an input stream of bytes, the encoding function streams the input into a compression function chained to the encryption function:

```
encode(key, input) = return encrypt(key, compress(input))
```

Given an encryption key and an input stream of bytes, the decoding function streams the input into the decryption function chained to the decompression function.

```
decode(key, input) = return decompress(decrypt(key, input))
```


## Initialization of an encrypted repository

User creating the repository is prompted for a passphrase that is passed through an entropy-based strength-check to refuse weak ones.

We derive a **256-bits master key** for the repository from the supplied passphrase using the KDF.

In addition,
**a 32-bytes random block** is generated and encrypted with the master key.

KDF parameters, salt and encrypted block are then stored in the repository configuration which remains unencrypted as it is used by clients to initialize:

```go
type Configuration struct {
	DataAlgorithm   string      // AES-GCM-SIV
	SubKeyAlgorithm string      // AES-KW
	KDFParams       KDFParams
	Canary          []byte
}

type KDFParams struct {
	KDF            string           // ARGON2ID by default, determines which params field is set below
	Salt           []byte           // randomly-generated

	Argon2IDParams *Argon2IDParams
	ScryptParams   *ScryptParams
	Pbkdf2Params   *PBKDF2Params
}

type Argon2IDParams struct {
	SaltSize uint32     // 128-bits salt
	Time     uint32     // 4
	Memory   uint32     // 256M
	Threads  uint8      // NumCPU
	KeyLen   uint32     // 256-bits
}
```

The configuration is stored in the repository using the **storage object wrapping format** described later in this document.
That wrapping format essentially prepends a small header and appends the MAC of header+content.

As such, the configuration is MAC-protected and a MAC mismatch leads to detection of a corruption:

```
$ plakar ls
repository passphrase: 
2025-02-07T00:27:33Z   873441e1    3.1 MB        0s /private/etc
$ 

[... editing the raw file to change a value ...]

$ plakar ls
repository passphrase: 
plakar: hmac mismatch
$
```



## Internal structure of a repository

A repository consists of three main elements:

- the configuration created during repository initialization
- packfiles
- state files

These are all wrapped in the **storage object wrapping format** described later in this document,
providing MAC integrity check.


### Configuration
The configuration supposedly contains no sensitive informations.

In addition to the crypto parameters described above,
it contains a UUID, a timestamp, the chunking settings for deduplication, the settings for compression (algorithm, level, ...).

```go
type Configuration struct {
	Version      versioning.Version
	Timestamp    time.Time
	RepositoryID uuid.UUID

	Packfile    packfile.Configuration
	Chunking    chunking.Configuration
	Hashing     hashing.Configuration
	Compression *compression.Configuration
	Encryption  *encryption.Configuration
}
```

It is serialized in binary format using the `msgpack` serializer and remains unencrypted in the repository.


### Packfiles
Packfiles are immutable structured files that are created during the backup process and which consist of three sections:

```
[DATA]
    [encoded blob #1]
    [encoded blob #2]
    [encoded blob #3]
    [...]

[ENCODED INDEX]
    [blob #1 type] [MAC(data #1, masterKey)] [offset] [length]
    [blob #2 type] [MAC(data #2, masterKey)] [offset] [length]
    [blob #3 type] [MAC(data #3, masterKey)] [offset] [length]
    [...]

[ENCODED FOOTER]
    Format version, creation timestamp, index MAC, index offset and number of records
```

#### Data section
The data section contains a sequence of blobs that were created during the backup process by individually encoding chunks of data of varying types and sizes.

These blobs are written one after the other in the packfile with no delimitation marker so that it's not possible to determine where they begin and where they end: the data section is just a stream of apparently random bytes.
They are individually encoded so it is possible to seek to a location in the packfile and decode a specific blob without requiring to read preceding blobs to build a compression or encryption state.

Since blobs are created from a variety of data structures that are not only related to file content but also to filesystem structure and software internal structures,
and because the backup process parallelizes packfiles filling, filesystem discovery and file content chunking,
two contiguous blobs in a packfile almost never represent data that is contiguous in the original data source being backed up.
A file content has blobs spread across multiple packfiles and two blobs from the same file content within a same packfile are almost always interlaced with unrelated blobs encoded from other data content.


#### Index section
The index section contains a sequence of fixed-length records that contain the blob type, the blob version, a MAC of the original data using the repository master key as secret, the offset of the blob in the data section and its length.

The packfile index is individually encoded and not meant to be used as the repository state files provide a global index that aggregates all packfiles indexes.
They are mainly used as a replica for disaster recovery should repository state end up corrupted:
if really needed, we can locate the offset of the index thanks to the packfile footer and decode it to access its content and understand the structure of the packfile.

The index has its MAC computed and stored in the footer for fast integrity check without needing to read the entire packfile.


#### Footer section
The footer is a fixed-size structure which allows to locate the offset of the index, its size and its MAC.

It does not contain sensitive data but is encrypted as to not leak the index offset and make it harder to determine the number of blobs in a packfile:
a packfile of ~20MB may contain only 2 x ~10MB blobs or may contain 20x ~1MB blobs, only the index provides this information.


### State files
 
State files are append-only structured files that provide an event log of changes happening to a repository.

Every time a backup is done,
a new state file is created that contains records describing which packfiles were created,
which blob MACs point to which locations (packfile, offset, length),
which snapshots were deleted (to allow a maintenance job to find orphaned packfiles to be removed).

The state files contain fixed records that look as follow:

```
[...]
[ET_LOCATIONS] [RECORD SIZE] [MAC(data #1, masterKey)] [packfile #3 identifier] [offset] [length]
[ET_LOCATIONS] [RECORD SIZE] [MAC(data #2, masterKey)] [packfile #1 identifier] [offset] [length]
[ET_LOCATIONS] [RECORD SIZE] [MAC(data #3, masterKey)] [packfile #2 identifier] [offset] [length]
[ET_PACKFILE]  [RECORD SIZE] [packfile #1 identifier]
[ET_PACKFILE]  [RECORD SIZE] [packfile #2 identifier]
[ET_PACKFILE]  [RECORD SIZE] [packfile #3 identifier]
[...]
```

There's no sensitive data in state files,
they are essentially a superset of packfile indexes.

The state files are encoded as a whole and not meant to be consumed directly by a client but fetched and synchronized in a local database.


## Repository objects wrapping format

The objects wrapping format is a structure format to wrap configuration, packfiles and state files with a common header and footer.

The purpose is to provide the following features:
- ability to detect the type of repository object in case state files and packfiles were accidentally mixed
- ability to detect the version of an object to properly deserialize state files and packfiles should multiple versions coexist
- ability to perform a MAC validation of the content

The format wraps a repository object as follows:

```
[MAGIC]             = "_PLAKAR_"
[TYPE]              = LE uint32() : object type (CONFIG=0, PACKFILE=1, STATE=2)
[VERSION]           = LE uint32() : object-specific version x.y.z encoded as (x << 24 | y << 8 | z )

    [OBJECT DATA]   = cleartext CONFIG | PACKFILE with encoded index and footer | encoded STATE

[MAC]              = MAC from all previous bytes including MAGIC, TYPE and VERSION
```

Whenever a wrapped object (config, packfiles and state files) is fetched from the repository,
and before passing the object data to upper layer for deserialization,
the wrapping is validated to ensure expectations are met:
a packfile loading for exampole should have the proper magic and type, a recognized version and a valid MAC.


## Opening of an encrypted repository

- The client retrieves the repository configuration and obtains the KDF parameters, the salt as well as the encrypted random 32-bytes block.

- It prompts user for the passphrase and derives the master key using the KDF parameters and salt.

- It verifies that the derived master key successfully decrypts the canary block using the data encryption algorithm integrity check:

- If decryption works, master key is used during the session, otherwise client errors out due to repository passphrase mismatch.

```
$ plakar ls
repository passphrase: 
2025-02-07T00:27:33Z   873441e1    3.1 MB        0s /private/etc
$ 
```

```
$ plakar ls
repository passphrase: 
repository passphrase: 
repository passphrase: 
plakar: could not derive secret
$ 
```


## State synchronization

Everytime a client opens a repository, regardless of the operation being performed, a state synchronization is performed to keep a local cache updated.

The client requests the repository for a list of state identifiers,
fetches the states that are absent from its local cache and decodes them to update the cache accordingly.

When the synchronization is done,
the local cache has a full representation of the repository state.
It can be used to check existence or location of blobs within the repository without having to query the repository directly for packfile and packfile indexes.

It is technically possible to check if a local filesystem exists within a repository without emiting a single query to the repository.


## Snapshots
A snapshot is essentially a virtual filesystem that describes the layout of directories entries, file entries and file data among a few other things.
Each node of the virtual filesystem is either a structure that points to other structures or to chunks of raw data.

All these nodes are stored as blobs within packfiles,
and pointers within the virtual filesystems are represented as MACs that can be looked up into packfiles to fetch related blobs.

Each snapshot is assigned a random 256-bits identifier that points to a virtual filesystem root,
and browsing the snapshot consists in resolving children nodes from there by performing the proper packfile lookups and fetches.

```
$ plakar ls e8b104b9:/private/etc     
repository passphrase: 
2024-12-07T08:11:55Z drwxr-xr-x                      160 B uucp

$ ./plakar ls e8b104b9:/private/etc/uucp
repository passphrase: 
2024-12-07T08:11:55Z -r--r--r--                      133 B passwd
2024-12-07T08:11:55Z -r--r--r--                      141 B port
2024-12-07T08:11:55Z -r--r--r--                     2.4 kB sys
$
```

### The backup process

```
$ plakar -trace repository,snapshot,packer backup /private/etc/uucp
[...]
```

An empty state is created to hold changes to the current local cache.

As backup progresses, chunks of data are produced from various sources:
file content, filesystem structure, plakar internal structures, ...

Each time a chunk is produced,
a MAC of the data is computed from the chunk and master key.
To determine if the chunk already exists in the repository or if it needs to be pushed,
that MAC is then used as a key to query local cache:

```
found = localCache.BlobExists(blobType, MAC(chunk, masterKey))
```

If found, then a blob with the same MAC already exists in the repository and there is no need to push it again.

If blob is not found,
then chunk is encoded:

```
blob = encode(masterKey, chunk)      // reminder: compress and encrypt with masterKey
```

and passed to a packing job that will append it to the data section of one of the packfiles currently being created, at random, using the blobId as its key:

```
trace: snapshot: b69919a9: CheckBlob(chunk, 17294d602f2d28944e6517a6a8a432548351d1eaf468062b8da6d84bbf7c5440)
trace: snapshot: b69919a9: PutBlob(chunk, 17294d602f2d28944e6517a6a8a432548351d1eaf468062b8da6d84bbf7c5440) len=133
trace: snapshot: b69919a9: CheckBlob(chunk, 863f779b43680f81799688e91c18a164047a1a8e9dfff650881e50ba35530ed1)
trace: snapshot: b69919a9: PutBlob(chunk, 863f779b43680f81799688e91c18a164047a1a8e9dfff650881e50ba35530ed1) len=141
trace: snapshot: b69919a9: CheckBlob(chunk, b86ff58053e9e930a0324f7fb5e0458213567feca9e97de92da20bff82f17e06)
trace: snapshot: b69919a9: PutBlob(chunk, b86ff58053e9e930a0324f7fb5e0458213567feca9e97de92da20bff82f17e06) len=2422
trace: packer: b69919a9: PackerMsg(4, 1.0.0, 3ccd81aafd51f67d699dd21bae5c33a4fd4d4849f1f0e291df9afe3884455f2b), dt=99.792µs
trace: packer: b69919a9: PackerMsg(4, 1.0.0, c9d3e886eccdf1472cacc0df812416adafbf28ab26df02eb1d51f3b5d4bb3a99), dt=19.584µs
trace: snapshot: b69919a9: CheckBlob(object, b86ff58053e9e930a0324f7fb5e0458213567feca9e97de92da20bff82f17e06)
trace: packer: b69919a9: PackerMsg(4, 1.0.0, bd37521333dc69efbd080e231e9f858167276b435c5004f8e9c6c45a59a9de06), dt=58.209µs
trace: snapshot: b69919a9: PutBlob(object, b86ff58053e9e930a0324f7fb5e0458213567feca9e97de92da20bff82f17e06) len=237
trace: snapshot: b69919a9: CheckBlob(object, 17294d602f2d28944e6517a6a8a432548351d1eaf468062b8da6d84bbf7c5440)
trace: snapshot: b69919a9: PutBlob(object, 17294d602f2d28944e6517a6a8a432548351d1eaf468062b8da6d84bbf7c5440) len=237
trace: snapshot: b69919a9: CheckBlob(object, 863f779b43680f81799688e91c18a164047a1a8e9dfff650881e50ba35530ed1)
trace: snapshot: b69919a9: PutBlob(object, 863f779b43680f81799688e91c18a164047a1a8e9dfff650881e50ba35530ed1) len=237
trace: snapshot: b69919a9: PutBlob(vfs, 02115b0348f69e0f922079e2c809ef61cc4fb00d7ac9beec192bbcc940bdd888) len=454
trace: packer: b69919a9: PackerMsg(5, 1.0.0, c9d3e886eccdf1472cacc0df812416adafbf28ab26df02eb1d51f3b5d4bb3a99), dt=17.917µs
trace: packer: b69919a9: PackerMsg(5, 1.0.0, 3ccd81aafd51f67d699dd21bae5c33a4fd4d4849f1f0e291df9afe3884455f2b), dt=18.375µs
trace: snapshot: b69919a9: PutBlob(vfs, 32f71c2b0b81718438d781abdade67f8cca76d85f4b5704c0ffc1f99e02c8976) len=457
trace: snapshot: b69919a9: PutBlob(vfs, b57d8719dbb3e49a049b048b19945061e2b585c190c9182dd0b6c9b7b768bf6e) len=455
trace: packer: b69919a9: PackerMsg(5, 1.0.0, bd37521333dc69efbd080e231e9f858167276b435c5004f8e9c6c45a59a9de06), dt=13.709µs
trace: packer: b69919a9: PackerMsg(6, 1.0.0, da7e435310c01b138fc5be4acac9a4d17731e67f1aa3909598575bfb606d1dcb), dt=24.708µs
trace: packer: b69919a9: PackerMsg(6, 1.0.0, 32ad79b323d46bc6169abe7d90169dffc03107f969703b9170c004223d3d9d66), dt=9.292µs
trace: packer: b69919a9: PackerMsg(6, 1.0.0, 1377869ce7827bbdbf59d49a2483841ce28c3c95ffc76f41f40c2e48c41b7e1e), dt=3.458µs
trace: snapshot: b69919a9: CheckBlob(error, 41e32c52c18dace8f5bc87c41649918e3bb3f1db46ceeb68164fb09314e7b842)
trace: snapshot: b69919a9: PutBlob(error, 41e32c52c18dace8f5bc87c41649918e3bb3f1db46ceeb68164fb09314e7b842) len=38
trace: snapshot: b69919a9: CheckBlob(error, 2fb46ff4617bd7ccb63f12bbb3e9e4c8a1252bc45750d94cb360813b960053de)
trace: snapshot: b69919a9: PutBlob(error, 2fb46ff4617bd7ccb63f12bbb3e9e4c8a1252bc45750d94cb360813b960053de) len=67
trace: packer: b69919a9: PackerMsg(11, 1.0.0, 173120dedf94a3b42c4c79f6760677e964bbc6bc81b60af2f883aad768489a6c), dt=13.75µs
trace: packer: b69919a9: PackerMsg(11, 1.0.0, c2d1b7f7c2caf6f796d02737f1a30c363e668b59ace775e30aa6c816584b77a0), dt=14.167µs
trace: snapshot: b69919a9: CheckBlob(vfs entry, 0391ff91cc70e1ff4c5f9639886bdf8ddacea5d90134a779c44a2fc6d8b62aa3)
trace: snapshot: b69919a9: PutBlob(vfs entry, 0391ff91cc70e1ff4c5f9639886bdf8ddacea5d90134a779c44a2fc6d8b62aa3) len=404
trace: snapshot: b69919a9: CheckBlob(vfs entry, 6329cbe414bd797cb9702a1e5822a53fe74e5aeacd21ceabe528af574fe3ce3f)
trace: snapshot: b69919a9: PutBlob(vfs entry, 6329cbe414bd797cb9702a1e5822a53fe74e5aeacd21ceabe528af574fe3ce3f) len=414
trace: packer: b69919a9: PackerMsg(7, 1.0.0, b84be23da3f9bacdaecf5d3be8cf155f97f32c26c0513bd004e171eb12c9793b), dt=9.666µs
trace: snapshot: b69919a9: CheckBlob(vfs entry, c3606a208c85b3c380714c4e27a27f6b3d465eca7e7e0d5c2bb178692a4d3b23)
trace: snapshot: b69919a9: PutBlob(vfs entry, c3606a208c85b3c380714c4e27a27f6b3d465eca7e7e0d5c2bb178692a4d3b23) len=478
trace: packer: b69919a9: PackerMsg(7, 1.0.0, 262535e7f2ccb68e8e3648721129583ee21dcccc4563037f07dda52560f6103b), dt=17.125µs
trace: snapshot: b69919a9: CheckBlob(vfs entry, 52ec4a80d3a87dea685025e4bf1b4c6cf43e47100e3cc8aeaebe20b008cf9de0)
trace: packer: b69919a9: PackerMsg(7, 1.0.0, 700970f6b2e3f90ada9cf9116fa9af7c6f908d5502d1858d0820509d3201cf2d), dt=13.084µs
trace: snapshot: b69919a9: PutBlob(vfs entry, 52ec4a80d3a87dea685025e4bf1b4c6cf43e47100e3cc8aeaebe20b008cf9de0) len=521
trace: snapshot: b69919a9: CheckBlob(vfs entry, 32f71c2b0b81718438d781abdade67f8cca76d85f4b5704c0ffc1f99e02c8976)
trace: snapshot: b69919a9: PutBlob(vfs entry, 32f71c2b0b81718438d781abdade67f8cca76d85f4b5704c0ffc1f99e02c8976) len=457
trace: packer: b69919a9: PackerMsg(7, 1.0.0, 32cb8b881f99810bf879351ae8b5aab8521c05237e26a5d6fd1b0688d9fceffc), dt=12µs
trace: snapshot: b69919a9: CheckBlob(vfs entry, b57d8719dbb3e49a049b048b19945061e2b585c190c9182dd0b6c9b7b768bf6e)
trace: snapshot: b69919a9: PutBlob(vfs entry, b57d8719dbb3e49a049b048b19945061e2b585c190c9182dd0b6c9b7b768bf6e) len=455
trace: packer: b69919a9: PackerMsg(7, 1.0.0, da7e435310c01b138fc5be4acac9a4d17731e67f1aa3909598575bfb606d1dcb), dt=9.791µs
trace: snapshot: b69919a9: CheckBlob(vfs entry, 02115b0348f69e0f922079e2c809ef61cc4fb00d7ac9beec192bbcc940bdd888)
trace: snapshot: b69919a9: PutBlob(vfs entry, 02115b0348f69e0f922079e2c809ef61cc4fb00d7ac9beec192bbcc940bdd888) len=454
trace: packer: b69919a9: PackerMsg(7, 1.0.0, 32ad79b323d46bc6169abe7d90169dffc03107f969703b9170c004223d3d9d66), dt=7.375µs
trace: packer: b69919a9: PackerMsg(7, 1.0.0, 1377869ce7827bbdbf59d49a2483841ce28c3c95ffc76f41f40c2e48c41b7e1e), dt=9.084µs
trace: snapshot: b69919a9: CheckBlob(vfs, cb3ebbc35d6d62864e21679a150070b4c51ab75919b9c120ea9ed6752f8ffe68)
trace: snapshot: b69919a9: PutBlob(vfs, cb3ebbc35d6d62864e21679a150070b4c51ab75919b9c120ea9ed6752f8ffe68) len=388
trace: snapshot: b69919a9: CheckBlob(vfs, 34c73d8858a6efcca25ba3fd8cc6f637e1c890137f5919cab836ed6fe7434195)
trace: snapshot: b69919a9: PutBlob(vfs, 34c73d8858a6efcca25ba3fd8cc6f637e1c890137f5919cab836ed6fe7434195) len=67
trace: packer: b69919a9: PackerMsg(6, 1.0.0, ae930663b00e784965828920b477dfb1bf90b50744e6d47c1fab7d315476408b), dt=11.083µs
trace: packer: b69919a9: PackerMsg(6, 1.0.0, 01c93591fe9beed73022b19b2286d91f8ec0863bd5e52fe8c138920cf3f4452b), dt=9.292µs
trace: snapshot: b69919a9: PutBlob(snapshot, b69919a9d695334d13ea961c76c66038217c69b380ca0dab1741f6bfca44b905) len=1054
trace: packer: b69919a9: PackerMsg(3, 1.0.0, b69919a9d695334d13ea961c76c66038217c69b380ca0dab1741f6bfca44b905), dt=12.125µs
$ 
```

The packfile index is updated to track the new blob:

```
[blob type] [blob version] [MAC(chunk, masterKey)] [blob offset] [blob length]
```

When the packfile reaches a certain threshold (~20MB as of today's configuration), it is serialized as described previously and pushed to the repository with a random packfile identifier.

```
trace: repository: PutPackfile(9f68c36079f44a46b0c89edc23a245cc24ab0bc8a0e980770c9e954c93a25f02, ...): 830.084µs
trace: repository: PutPackfile(faca8f6ae948f4fb8fe5e7e961c07d8bc1626947bab673a3dad8702f51163d6e, ...): 928.125µs
trace: repository: PutPackfile(034093b741e5bdbd0059db71f49e963271c2cb6a994d9f28943340b5d819b81d, ...): 280.833µs
trace: repository: PutPackfile(e73aab7dc996fab95bfa262110b43e20afa1465fde3585ee46b601780c5b02b2, ...): 963.5µs
trace: repository: PutPackfile(6bbd59007dc25b5093c4cdbf994097560d327c4da7fb74e65740fd13943a4ea3, ...): 1.020708ms
trace: repository: PutPackfile(e6c0f5638a8166bebfbcdab2c2d060dfc6ac76f95e4e5a4c0d36247223948f0e, ...): 858.125µs
trace: repository: PutPackfile(ed9c0d4b5dd910d1f09810e792025d01b7352c68c0149c6a0a3ada0b48417601, ...): 1.077208ms
trace: repository: PutPackfile(b95161ecee260edb0cd58eb84de4161711bc49099e9828272a1652ed992a46b5, ...): 848.75µs
trace: repository: PutPackfile(09f6323ea8e14abab0263b703e0fea6acdef0ec9c36dbe4dfce1e79acd22909e, ...): 1.0515ms
trace: repository: PutPackfile(b1bb2f1b792541114ed8f9a4ad97d1dd7ca39c58a3f412d8387cc3ab2f9418f6, ...): 1.006292ms
trace: repository: PutPackfile(fe47dc56264b9923e261ed08c642fcfbae679b9cd65be11fa9001223b6f54624, ...): 953.791µs
trace: repository: PutPackfile(e2188140e524678626431499ae487efce79820c98a8df9e4e257b0c6a8990f4f, ...): 1.066917ms
trace: repository: PutPackfile(6bb84eb9a55d34f9f1c7fe4015284435723dcc6a570cb524bdad3db41736ef9e, ...): 1.228375ms
trace: repository: PutPackfile(6087ca01172cecabc378633f9e7426ebeae96e7f07fb64784d64ebbd1d590ec1, ...): 1.207459ms
```

The snapshot state is then updated to reflect the creation of a packfile and map the blob to a packfile location by replicating the packfile index entry describe above.

When there are no more chunks to process,
the snapshot state is encoded:
```
state = encode(masterKey, state)
```

then pushed to the repository making the changes visible to clients and allowing them to update their local caches:

```
trace: repository: PutState(bda0f87c0b16a96ff44373796c14163ef256e69c777e393f27847a5e17db3c33, ...): 188.417µs
```

Note again that both PutPackfile and PutState wrap the data in the **repository objects wrapping format**.


### The restore process

```
$ ./plakar -trace all restore bda0f87c
repository passphrase: 
```

Given a snapshot identifier,
the corresponding blob for a snapshot header is fetched (possibly from local cache) and decoded:

```
trace: snapshot: repository.GetSnapshot(bda0f87c0b16a96ff44373796c14163ef256e69c777e393f27847a5e17db3c33)
trace: repository: Decode: 5.459µs
trace: repository: Decode(997 bytes): 97.917µs
```

From there,
the virtual filesystem tied to the snapshot can then be traversed with each node being fetched and decoded on the fly from the MAC found in each node.

```
trace: repository: GetPackfileBlob(6bbd59007dc25b5093c4cdbf994097560d327c4da7fb74e65740fd13943a4ea3, 282, 174): 891.875µs
trace: repository: GetBlob(vfs, 5fb6a97041326a45ca5811ee0da91e92dd016747cd52e6e8d5dd83b0d43018b4): 902.833µs
trace: repository: Decode: 3.458µs
trace: repository: Decode(440 bytes): 85.584µs
trace: repository: GetPackfileBlob(6bb84eb9a55d34f9f1c7fe4015284435723dcc6a570cb524bdad3db41736ef9e, 477, 440): 127.375µs
trace: repository: GetBlob(vfs, e3cf3cfd4e50bf53384d9a733d75a0b78f9ae5a233bd91a5f41efc15ea48dc3f): 138.417µs
trace: repository: Decode: 3.167µs
trace: repository: Decode(443 bytes): 526.875µs
trace: repository: GetPackfileBlob(e73aab7dc996fab95bfa262110b43e20afa1465fde3585ee46b601780c5b02b2, 0, 443): 557.625µs
trace: repository: GetBlob(vfs entry, 351ba23aa7e783747829f4f204e3a4cdb1b4ecc2b86a82ce0e284702896373ef): 565µs
trace: repository: Decode: 2.959µs
trace: repository: Decode(440 bytes): 97.833µs
trace: repository: GetPackfileBlob(6bb84eb9a55d34f9f1c7fe4015284435723dcc6a570cb524bdad3db41736ef9e, 477, 440): 117.583µs
trace: repository: GetBlob(vfs, e3cf3cfd4e50bf53384d9a733d75a0b78f9ae5a233bd91a5f41efc15ea48dc3f): 126.584µs
trace: repository: Decode: 9.917µs
trace: repository: Decode(457 bytes): 499.916µs
```

Actual data is located in leaf nodes and can be restored through the same process of blob fetching and decoding,
then writing to a local filesystem file:

```
$ ./plakar restore e8b104b9:/private/etc/uucp
repository passphrase: 
info: e8b104b9: OK ✓ /private/etc/uucp/passwd
info: e8b104b9: OK ✓ /private/etc/uucp/port
info: e8b104b9: OK ✓ /private/etc/uucp/sys
info: e8b104b9: OK ✓ /private/etc/uucp
$ 
```

Integrity of a restore can be verified as MACs can be recomputed for each blob fetched from the repository and compared to the virtual filesystem recorded ones.


## Encryption and decryption functions

Data is encrypted using individual 256-bits subkeys that are randomly-generated for each blob and encrypted themselves with the master key.

The purpose is to ensure that should a subkey be broken or leak for reasons unrelated to the master key being compromised,
only the related blob would be at risk and not the entire repository.

For that reason, plakar uses a subkey encryption algorithm and a data encryption algorithm, each operating on a specific phase of the encryption/decryption process.

### Encryption

The encryption works as follows:

    1- an encryption function takes a master key and an input buffer
    2- it randomly generates a 256-bits subkey
    3- a subkey encryption function is called to produce an encrypted subkey block protected by the master key
    4- it prepends that encrypted prefix block to output
    5- it then encrypts the data in chunks of 64K using the data encryption function with subkey

    [encrypted prefix block] [encrypted data block]
     ^^^^^^^^^^^^^^^^^^^^^^   ^^^^^^^^^^^^^^^^^^^^
     master key encrypted     subkey encrypted


### Decryption

The decryption works as follows:

    1- a decryption function takes a master key and an input buffer
    2- the input buffer is split into two parts: the prefix block and the data block
    3- the prefix block is decrypted so that subkey is retrieved
    4- the data block is decrypted in chunks of 64K with the subkey, GCM integrity check validates subkey
