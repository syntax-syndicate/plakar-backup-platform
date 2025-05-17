/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION
 * OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN
 * CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package ptar

import (
	"bytes"
	"flag"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/google/uuid"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Ptar{} }, subcommands.BeforeRepositoryWithStorage, "ptar")
}

type listFlag []string

func (l *listFlag) String() string {
	return fmt.Sprint([]string(*l))
}

func (l *listFlag) Set(value string) error {
	for _, v := range *l {
		if v == value {
			return nil
		}
	}
	*l = append(*l, value)
	return nil
}

func (cmd *Ptar) Parse(ctx *appcontext.AppContext, args []string) error {

	cmd.KlosetUUID = uuid.Must(uuid.NewRandom())

	flags := flag.NewFlagSet("ptar", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: plakar %s [OPTIONS] file.ptar [FILES]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&cmd.AllowWeak, "weak-passphrase", false, "allow weak passphrase to protect the repository")
	flags.StringVar(&cmd.Hashing, "hashing", hashing.DEFAULT_HASHING_ALGORITHM, "hashing algorithm to use for digests")
	flags.BoolVar(&cmd.NoEncryption, "no-encryption", false, "disable transparent encryption")
	flags.BoolVar(&cmd.NoCompression, "no-compression", false, "disable transparent compression")
	flags.Var(&cmd.SyncTargets, "sync", "add a kloset location to include in the ptar archive (can be specified multiple times)")
	flags.Var(&cmd.BackupTargets, "backup", "add a backup location to include in the ptar archive (can be specified multiple times)")
	flags.Parse(args)

	if flags.NArg() == 0 {
		cmd.KlosetPath = fmt.Sprintf("%s.ptar", cmd.KlosetUUID)
	} else if flags.NArg() == 1 {
		cmd.KlosetPath = flags.Arg(0)
	} else {
		return fmt.Errorf("%s: too many parameters", flag.CommandLine.Name())
	}

	for _, syncTarget := range cmd.SyncTargets {
		var peerSecret []byte

		storeConfig, err := ctx.Config.GetRepository(syncTarget)
		if err != nil {
			return fmt.Errorf("peer repository: %w", err)
		}

		peerStore, peerStoreSerializedConfig, err := storage.Open(ctx, storeConfig)
		if err != nil {
			return err
		}

		peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
		if err != nil {
			return err
		}

		if peerStoreConfig.Encryption != nil {
			if pass, ok := storeConfig["passphrase"]; ok {
				key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(pass))
				if err != nil {
					return err
				}
				if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
					return fmt.Errorf("invalid passphrase")
				}
				peerSecret = key
			} else {
				for {
					passphrase, err := utils.GetPassphrase("source repository")
					if err != nil {
						fmt.Fprintf(os.Stderr, "%s\n", err)
						continue
					}

					key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, passphrase)
					if err != nil {
						return err
					}
					if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
						return fmt.Errorf("invalid passphrase")
					}
					peerSecret = key
					break
				}
			}
		}

		peerCtx := appcontext.NewAppContextFrom(ctx)
		peerCtx.SetSecret(peerSecret)
		_, err = repository.NewNoRebuild(peerCtx, peerStore, peerStoreSerializedConfig)
		if err != nil {
			return err
		}
		cmd.SyncSecrets = append(cmd.SyncSecrets, peerSecret)
	}

	if hashing.GetHasher(strings.ToUpper(cmd.Hashing)) == nil {
		return fmt.Errorf("%s: unknown hashing algorithm", flag.CommandLine.Name())
	}

	return nil
}

type Ptar struct {
	subcommands.SubcommandBase

	KlosetPath string
	KlosetUUID uuid.UUID

	AllowWeak     bool
	Hashing       string
	NoEncryption  bool
	NoCompression bool

	SyncTargets   listFlag
	SyncSecrets   [][]byte
	BackupTargets listFlag
}

func (cmd *Ptar) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	storageConfiguration := storage.NewConfiguration()
	storageConfiguration.RepositoryID = cmd.KlosetUUID

	if cmd.NoCompression {
		storageConfiguration.Compression = nil
	} else {
		storageConfiguration.Compression = compression.NewDefaultConfiguration()
	}

	hashingConfiguration, err := hashing.LookupDefaultConfiguration(strings.ToUpper(cmd.Hashing))
	if err != nil {
		return 1, err
	}
	storageConfiguration.Hashing = *hashingConfiguration

	minEntropBits := 80.
	if cmd.AllowWeak {
		minEntropBits = 0.
	}

	var hasher hash.Hash
	if !cmd.NoEncryption {
		storageConfiguration.Encryption = encryption.NewDefaultConfiguration()

		var passphrase []byte

		envPassphrase := os.Getenv("PLAKAR_PASSPHRASE")
		if ctx.KeyFromFile == "" {
			if envPassphrase != "" {
				passphrase = []byte(envPassphrase)
			} else {
				for attempt := 0; attempt < 3; attempt++ {
					tmp, err := utils.GetPassphraseConfirm("ptar", minEntropBits)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%s\n", err)
						continue
					}
					passphrase = tmp
					break
				}
			}
		} else {
			passphrase = []byte(ctx.KeyFromFile)
		}

		if len(passphrase) == 0 {
			return 1, fmt.Errorf("can't encrypt the repository with an empty passphrase")
		}

		key, err := encryption.DeriveKey(storageConfiguration.Encryption.KDFParams, passphrase)
		if err != nil {
			return 1, err
		}

		canary, err := encryption.DeriveCanary(storageConfiguration.Encryption, key)
		if err != nil {
			return 1, err
		}
		storageConfiguration.Encryption.Canary = canary
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, key)
		ctx.SetSecret(key)
	} else {
		storageConfiguration.Encryption = nil
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

	storageConfiguration.Packfile.MaxSize = math.MaxUint64

	serializedConfig, err := storageConfiguration.ToBytes()
	if err != nil {
		return 1, err
	}

	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	if err != nil {
		return 1, err
	}
	wrappedConfig, err := io.ReadAll(rd)
	if err != nil {
		return 1, err
	}

	st, err := storage.Create(ctx, map[string]string{"location": "ptar:" + cmd.KlosetPath}, wrappedConfig)
	if err != nil {
		return 1, err
	}

	repo, err = repository.New(ctx, st, wrappedConfig)
	if err != nil {
		return 1, err
	}

	identifier := objects.RandomMAC()
	scanCache, err := repo.AppContext().GetCache().Scan(identifier)
	if err != nil {
		return 1, err
	}

	repoWriter := repo.NewRepositoryWriter(scanCache, identifier, repository.PtarType)
	for i, syncTarget := range cmd.SyncTargets {
		storeConfig, err := ctx.Config.GetRepository(syncTarget)
		if err != nil {
			return 1, fmt.Errorf("source repository: %w", err)
		}

		peerStore, peerStoreSerializedConfig, err := storage.Open(ctx, storeConfig)
		if err != nil {
			return 1, fmt.Errorf("could not open source store %s: %s", syncTarget, err)
		}

		srcCtx := appcontext.NewAppContextFrom(ctx)
		srcCtx.SetSecret(cmd.SyncSecrets[i])
		srcRepository, err := repository.New(srcCtx, peerStore, peerStoreSerializedConfig)
		if err != nil {
			return 1, fmt.Errorf("could not open source repository %s: %s", syncTarget, err)
		}

		if err := cmd.synchronize(ctx, srcRepository, repoWriter); err != nil {
			return 1, err
		}
	}
	if err := cmd.backup(ctx, repoWriter); err != nil {
		return 1, err
	}

	// We are done with everything we can now stop the backup routines.
	repoWriter.PackerManager.Wait()
	err = repoWriter.CommitTransaction(identifier)
	if err != nil {
		return 1, err
	}

	if err := st.Close(); err != nil {
		return 1, err
	}

	return 0, nil
}

func (cmd *Ptar) backup(ctx *appcontext.AppContext, repo *repository.RepositoryWriter) error {
	for _, loc := range cmd.BackupTargets {
		imp, err := importer.NewImporter(ctx, map[string]string{"location": loc})
		if err != nil {
			return err
		}

		snap, err := snapshot.CreateWithRepositoryWriter(repo)
		if err != nil {
			return err
		}

		backupOptions := &snapshot.BackupOptions{
			MaxConcurrency: 4,
			NoCheckpoint:   true,
			NoCommit:       true,
		}

		err = snap.Backup(imp, backupOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *Ptar) synchronize(ctx *appcontext.AppContext, srcRepository *repository.Repository, dstRepository *repository.RepositoryWriter) error {
	srcLocateOptions := utils.NewDefaultLocateOptions()
	srcSnapshotIDs, err := utils.LocateSnapshotIDs(srcRepository, srcLocateOptions)
	if err != nil {
		return err
	}

	for _, snapshotID := range srcSnapshotIDs {
		if err := ctx.Err(); err != nil {
			return err
		}

		srcSnapshot, err := snapshot.Load(srcRepository, snapshotID)
		if err != nil {
			return err
		}
		defer srcSnapshot.Close()

		dstSnapshot, err := snapshot.CreateWithRepositoryWriter(dstRepository)
		if err != nil {
			return err
		}
		defer dstSnapshot.Close()

		// overwrite the header, we want to keep the original snapshot info
		dstSnapshot.Header = srcSnapshot.Header

		if err := srcSnapshot.Synchronize(dstSnapshot); err != nil {
			return err
		}

		if err := dstSnapshot.Commit(nil, false); err != nil {
			return err
		}
	}

	return nil
}
