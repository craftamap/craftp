/**
* current local directory for tests:
* /home/fabian/craftpTest/
*
 */

package main

import (
	"fmt"
	"github.com/craftamap/craftp/utils"
	"github.com/pkg/sftp"
	"github.com/tkanos/gonfig"
	"golang.org/x/crypto/ssh"
	kh "golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	hostKeyFile, err := kh.New("/home/fabian/.ssh/known_hosts")
	if err != nil {
		fmt.Println("a")
	}

	configuration := Configuration{}
	err = gonfig.GetConf("/home/fabian/craftpTest/.craftp/config", &configuration)
	if err != nil {
		fmt.Println(err)
	}

	config := &ssh.ClientConfig{
		User:            configuration.User,
		HostKeyCallback: hostKeyFile,
	}
	config.Auth = []ssh.AuthMethod{}

	if utils.ContainsString(configuration.AuthMethod, "password") {
		config.Auth = append(config.Auth, ssh.Password(passwd()))
	}

	if utils.ContainsString(configuration.AuthMethod, "private_key") {
		key, err := ioutil.ReadFile("/home/fabian/.ssh/id_rsa")
		if err != nil {
			log.Fatalf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passwd()))
		if err != nil {
			log.Fatalf("unable to parse private key %v", err)
		}

		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	conn, err := ssh.Dial("tcp", configuration.Host+":"+strconv.Itoa(configuration.Port), config)
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Println(err)
	}
	defer client.Close()

	fmt.Println(client.Getwd())

	//Tree(client, []string{configuration.BaseDir}, 0)

}

func Tree(client *sftp.Client, basedir []string, count int) {
	path := client.Join(basedir...)

	filelist, _ := client.ReadDir(path)

	for _, v := range filelist {
		fmt.Println(strings.Repeat(" ", count) + v.Name())
		b := append([]string(nil), basedir...)
		b = append(b, v.Name())
		if v.IsDir() {
			Tree(client, b, count+1)
		}
	}
}

func passwd() string {
	fmt.Print("Enter Password:")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println(err)
	}
	password := string(bytePassword)
	fmt.Print("\n")
	return strings.TrimSpace(password)
}

type Configuration struct {
	User       string
	Host       string
	Port       int
	AuthMethod []string
	BaseDir    string
}
