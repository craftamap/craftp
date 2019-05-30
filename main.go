package main

import (
	"fmt"
	"github.com/craftamap/craftp/utils"
	"github.com/pkg/sftp"
	"github.com/tkanos/gonfig"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh"
	kh "golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

func main() {

	app := cli.NewApp()
	app.Name = "craftp"
	app.Version = "v0.1.0dev190530"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Fabian Siegel",
			Email: "kontakt@siegelfabian.de",
		},
	}
	app.HelpName = "craftp"
	app.Usage = "A git-like sftp client written in go"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  ".",
			Usage:  "path for a specific config directory from `FILE`",
			EnvVar: "CRAFTP_CONFIG",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "tree",
			Aliases: []string{"t"},
			Usage:   "prints a tree of the remote server",
			Action: func(c *cli.Context) error {
				configLocation := c.GlobalString("config")
				configuration := Configuration{}
				err := gonfig.GetConf(sftp.Join(configLocation, "config"), &configuration)
				if err != nil {
					fmt.Println(err)
				}
				conn := craftpConnection{}
				conn.Connect(configuration)
				defer conn.Close()
				conn.Tree([]string{configuration.BaseDir}, 0)
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	//Tree(client, []string{configuration.BaseDir}, 0)

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

type craftpConnection struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func (s *craftpConnection) Connect(configuration Configuration) {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	hostKeyFile, err := kh.New(sftp.Join(usr.HomeDir, ".ssh/known_hosts"))
	if err != nil {
		fmt.Println("a")
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
		key, err := ioutil.ReadFile(sftp.Join(usr.HomeDir, "/.ssh/id_rsa"))
		if err != nil {
			log.Fatalf("unable to read private key: %v", err)
		}

		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passwd()))
		if err != nil {
			log.Fatalf("unable to parse private key %v", err)
		}

		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	s.sshClient, err = ssh.Dial("tcp", configuration.Host+":"+strconv.Itoa(configuration.Port), config)
	if err != nil {
		fmt.Println(err)
	}

	s.sftpClient, err = sftp.NewClient(s.sshClient)
	if err != nil {
		fmt.Println(err)
	}

}

func (s *craftpConnection) Close() {
	s.sftpClient.Close()
	s.sshClient.Close()
}

func (s *craftpConnection) Tree(basedir []string, count int) {
	path := s.sftpClient.Join(basedir...)
	filelist, err := s.sftpClient.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range filelist {
		fmt.Println(strings.Repeat(" ", count) + v.Name())
		b := append([]string(nil), basedir...)
		b = append(b, v.Name())
		if v.IsDir() {
			s.Tree(b, count+1)
		}
	}
}
