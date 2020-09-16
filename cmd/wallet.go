package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/GGP1/kure/db"
	"github.com/GGP1/kure/wallet"

	"github.com/atotto/clipboard"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var walletCmd = &cobra.Command{
	Use:   "wallet <name> [-a add | -c copy | -d delete | -l list] [-t timeout]",
	Short: "Add, copy, delete or list wallets",
	Run: func(cmd *cobra.Command, args []string) {
		name := strings.Join(args, " ")

		if add {
			if err := addWallet(); err != nil {
				log.Fatal("error: ", err)
			}
			return
		}

		if copy {
			if err := copyWallet(name, timeout); err != nil {
				log.Fatal("error: ", err)
			}
		}

		if delete {
			if err := deleteWallet(name); err != nil {
				log.Fatal("error: ", err)
			}
			return
		}

		if err := listWallet(name); err != nil {
			log.Fatal("error: ", err)
		}

	},
}

func init() {
	RootCmd.AddCommand(walletCmd)
	walletCmd.Flags().BoolVarP(&add, "add", "a", false, "add a wallet")
	walletCmd.Flags().BoolVarP(&copy, "copy", "c", false, "copy wallet public key")
	walletCmd.Flags().BoolVarP(&delete, "delete", "d", false, "delete a wallet")
	walletCmd.Flags().BoolVarP(&list, "list", "l", true, "list wallet/wallets")
	walletCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0, "clipboard cleaning timeout")
}

func addWallet() error {
	wallet, err := walletInput()
	if err != nil {
		return err
	}

	if err := db.CreateWallet(wallet); err != nil {
		return err
	}

	fmt.Print("\nSucessfully created the wallet.")
	return nil
}

func copyWallet(name string, timeout time.Duration) error {
	wallet, err := db.GetWallet(name)
	if err != nil {
		return err
	}

	if err := clipboard.WriteAll(wallet.PublicKey); err != nil {
		return err
	}

	if timeout > 0 {
		<-time.After(timeout)
		clipboard.WriteAll("")
		os.Exit(1)
	}

	return nil
}

func deleteWallet(name string) error {
	_, err := db.GetWallet(name)
	if err != nil {
		return errors.New("This wallet does not exist")
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Are you sure you want to proceed? [y/n]: ")

	scanner.Scan()
	text := scanner.Text()
	input := strings.ToLower(text)

	if strings.Contains(input, "y") || strings.Contains(input, "yes") {
		if err := db.DeleteWallet(name); err != nil {
			log.Fatal("error: ", err)
		}

		fmt.Printf("\nSuccessfully deleted %s wallet.", name)
	}

	return nil
}

func listWallet(name string) error {
	if name != "" {
		wallet, err := db.GetWallet(name)
		if err != nil {
			return err
		}

		printWallet(wallet)
		return nil
	}

	wallets, err := db.ListWallets()
	if err != nil {
		return err
	}

	for _, wallet := range wallets {
		printWallet(wallet)
	}

	return nil
}

func walletInput() (*wallet.Wallet, error) {
	var name, wType, scriptT, keystoreT, seedPhrase, publicKey, privateKey string

	scanner := bufio.NewScanner(os.Stdin)

	name = scan(scanner, "Name", name)
	wType = scan(scanner, "Type", wType)
	scriptT = scan(scanner, "Script type", scriptT)
	keystoreT = scan(scanner, "Keystore Type", keystoreT)
	seedPhrase = scan(scanner, "Seed Phrase", seedPhrase)
	publicKey = scan(scanner, "Public Key", publicKey)
	privateKey = scan(scanner, "Private Key", privateKey)

	name = strings.ToLower(name)

	wallet := wallet.New(name, wType, scriptT, keystoreT, seedPhrase, publicKey, privateKey)

	return wallet, nil
}

func printWallet(w *wallet.Wallet) {
	w.Name = strings.Title(w.Name)

	if hide {
		w.SeedPhrase = ""
		w.PrivateKey = ""
	}

	str := fmt.Sprintf(
		`
+---------------+----------------->
| Name	        | %s
| Type      	| %s
| Script Type   | %s
| Keystore Type | %s
| Seed Phrase   | %s
| Public Key    | %s
| Private Key   | %s
+---------------+----------------->`,
		w.Name, w.Type, w.ScriptType, w.KeystoreType, w.SeedPhrase, w.PublicKey, w.PrivateKey)
	fmt.Println(str)
}