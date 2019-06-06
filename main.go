package main

import (
	"encoding/json"
	"errors"
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
	app.Version = "v0.1.0dev190605"
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
			Value:  "./.craftp",
			Usage:  "path for a specific config directory from `FILE`",
			EnvVar: "CRAFTP_CONFIG",
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "base, b",
			Value:  "",
			Usage:  "path for a specific base directory from `FILE`",
			EnvVar: "CRAFTP_BASE",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "tree",
			Aliases: []string{"t"},
			Usage:   "prints a tree of the remote server",
			Action: func(c *cli.Context) error {
				return errors.New("")
			},
		},
		{
			Name: "init",
			Action: func(c *cli.Context) error {
				var err error
				client := CraftpClient{}
				first := c.Args().Get(0)
				if first == "" {
					first = "."
				}
				err = client.Init(first)
				return err
			},
		},
		{
			Name: "remote",
			Action: func(c *cli.Context) error {
				client, err := CraftpClientNew(c.GlobalString("base"), c.GlobalString("config"))

				client.Remote()

				return err

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

type CraftpConfiguration struct {
	User       string
	Host       string
	Port       int
	AuthMethod []string
	BaseDir    string
}

type CraftpClient struct {
	connection    CraftpConnection
	configuration CraftpConfiguration
	local         CraftpLocal
}

func CraftpClientNew(baseDir string, configDir string) (CraftpClient, error) {
	local := CraftpLocal{
		baseDir:   baseDir,
		configDir: configDir,
	}
	wd, _ := os.Getwd()
	local.FindBase(wd + "/" + local.baseDir)
	local.configDir = baseDir + ".craftp"
	config, err := local.GetConfiguration()
	if err != nil {
		log.Fatal(err)
	}

	client := CraftpClient{
		local:         local,
		configuration: config,
	}

	return client, err
}

func (client *CraftpClient) Remote() error {
	fmt.Println("Your current remote server is:")
	fmt.Println("sftp://" + client.configuration.User + "@" + client.configuration.Host + ":" + strconv.Itoa(client.configuration.Port) + "/" + client.configuration.BaseDir)

	return nil
}

func (c *CraftpClient) Init(path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()
	wd, err := os.Getwd()
	filestat, err := file.Stat()
	if filestat.IsDir() && !c.local.HasBase(wd+"/"+path) {
		err = os.Mkdir(sftp.Join(wd, path, ".craftp"), os.ModeDir+0750)
		if err != nil {
			log.Fatal(err)
			return err
		}
		configFile, err := os.OpenFile(sftp.Join(path, ".craftp", "config"), os.O_CREATE|os.O_WRONLY, 0750)
		if err != nil {
			log.Fatal(err)
		}
		defer configFile.Close()
		emptyConf := CraftpConfiguration{}
		jsonConf, err := json.Marshal(emptyConf)
		if err != nil {
			log.Fatal(err)
			return err
		}
		_, err = configFile.Write(jsonConf)
		if err != nil {
			log.Fatal(err)
			return err
		}

	} else if c.local.HasBase(wd + "/" + path) {
		return errors.New("nesting not allowed or project already initialized")
	} else {
		return errors.New("path is a file or an unknown error occoured")
	}
	return err

}

type CraftpConnection struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func (s *CraftpConnection) Connect(configuration CraftpConfiguration) {
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

func (s *CraftpConnection) Close() {
	s.sftpClient.Close()
	s.sshClient.Close()
}

func (s *CraftpConnection) Tree(basedir []string, count int) {
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

type CraftpLocal struct {
	baseDir   string
	configDir string
}

func (c *CraftpLocal) FindBase(path string) (string, error) {
	base := ""
	for k, _ := range strings.Split(path, "/") {
		p := "/" + sftp.Join(strings.Split(path, "/")[:k+1][1:]...)

		file, err := os.Open(p)
		if err != nil {
			log.Fatal(err)
		}
		list, err := file.Readdir(0)
		if err != nil {
			log.Fatal(err)
		}
		for _, v := range list {
			if v.IsDir() && v.Name() == ".craftp" {
				base = p
			}

		}
	}

	if base != "" {
		c.baseDir = base
		return base, nil
	} else {
		return "", errors.New("no basedirectory")
	}

}

func (c *CraftpLocal) HasBase(path string) bool {
	_, err := c.FindBase(path)
	if err != nil {
		return false
	}
	return true
}

func (c *CraftpLocal) GetConfiguration() (CraftpConfiguration, error) {
	configuration := CraftpConfiguration{}
	err := gonfig.GetConf(sftp.Join(c.configDir, "config"), &configuration)
	if err != nil {
		fmt.Println(err)
		return configuration, err
	}
	return configuration, nil
}
