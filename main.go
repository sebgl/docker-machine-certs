package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/machine/drivers/generic"
	"github.com/docker/machine/libmachine/auth"
	"github.com/docker/machine/libmachine/cert"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/engine"
	"github.com/docker/machine/libmachine/host"
	"github.com/docker/machine/libmachine/persist"
	"github.com/docker/machine/libmachine/swarm"
	"github.com/docker/machine/libmachine/version"
)

var (
	outDir      = flag.String("out-dir", "out", "Path to the output certs directory")
	serverIP    = flag.String("server-ip", "", "Server IP (optional if DNS is defined)")
	serverDNS   = flag.String("server-dns", "", "Server DNS (optional if IP is defined)")
	machineName = flag.String("machine-name", "", "Machine name")
	sshKeyPath  = flag.String("ssh-key-path", "", "Path to ssh key to set in config.json")

	sshUser = flag.String("ssh-user", "root", "SSH User to set in config.json")
	sshPort = flag.Int("ssh-port", 22, "SSH port to set in config.json")
)

func checkFlags() {
	if *serverIP == "" {
		log.Fatal("--server-ip must be set")
	}
	if *machineName == "" {
		log.Fatal("--machine-name must be set")
	}
	if *sshKeyPath == "" {
		log.Fatal("--ssh-key-path must be set")
	}
}

/**********************
	File path utils
**********************/

func certsPath() string {
	return filepath.Join(*outDir, "certs")
}

func certsFile(name string) string {
	return filepath.Join(certsPath(), name)
}

func machinePath() string {
	return filepath.Join(*outDir, "machines", *machineName)
}

func machineFile(name string) string {
	return filepath.Join(machinePath(), name)
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func createDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
}

func setAbsolutePaths() {
	*outDir, _ = filepath.Abs(*outDir)
	*sshKeyPath, _ = filepath.Abs(*sshKeyPath)
}

/**********************
	Certs creation
**********************/

func BootstrapClientCert() error {
	// if ca.pem already exists, consider there's nothing to do
	if _, err := os.Stat(certsFile("ca.pem")); !os.IsNotExist(err) {
		fmt.Printf("Client certs found, skipping client cert creation\n")
		return nil
	}

	opts := &auth.Options{
		CertDir:          certsPath(),
		CaCertPath:       certsFile("ca.pem"),
		CaPrivateKeyPath: certsFile("ca-key.pem"),
		ClientCertPath:   certsFile("cert.pem"),
		ClientKeyPath:    certsFile("key.pem"),
	}

	err := cert.BootstrapCertificates(opts)
	if err != nil {
		return err
	}

	fmt.Println("Client cert files sucessfully created")

	return nil
}

func CreateServerCert(ip string, dns string) error {
	return cert.GenerateCert(
		[]string{ip, dns},
		machineFile("server.pem"),
		machineFile("server-key.pem"),
		machineFile("ca.pem"),
		certsFile("ca-key.pem"),
		*serverDNS,
		2048)
}

/**********************
	config.json
**********************/

func CreateConfigJSON() error {
	driver := &generic.Driver{
		EnginePort: engine.DefaultPort,
		SSHKey:     machineFile("id_rsa"),
		BaseDriver: &drivers.BaseDriver{
			MachineName: *machineName,
			StorePath:   machinePath(),
			IPAddress:   *serverIP,
			SSHUser:     *sshUser,
			SSHKeyPath:  *sshKeyPath,
			SSHPort:     *sshPort,
		},
	}
	host := &host.Host{
		ConfigVersion: version.ConfigVersion,
		Name:          driver.GetMachineName(),
		Driver:        driver,
		DriverName:    driver.DriverName(),
		HostOptions: &host.Options{
			AuthOptions: &auth.Options{
				CertDir:          certsPath(),
				CaCertPath:       certsFile("ca.pem"),
				CaPrivateKeyPath: certsFile("ca-key.pem"),
				ClientCertPath:   certsFile("cert.pem"),
				ClientKeyPath:    certsFile("key.pem"),
				ServerCertPath:   machineFile("server.pem"),
				ServerKeyPath:    machineFile("server-key.pem"),
				StorePath:        machinePath(),
			},
			EngineOptions: &engine.Options{
				InstallURL: drivers.DefaultEngineInstallURL,
				TLSVerify:  true,
			},
			SwarmOptions: &swarm.Options{
				Host:     "tcp://0.0.0.0:3376",
				Image:    "swarm:latest",
				Strategy: "spread",
			},
		},
	}
	filestore := persist.NewFilestore(*outDir, "", "")
	err := filestore.Save(host)
	if err != nil {
		return err
	}
	return nil
}

/**********************
		Main
**********************/

func main() {
	// parse flags
	flag.Parse()
	checkFlags()
	setAbsolutePaths()

	// create client cert if needed
	err := BootstrapClientCert()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// create server dir if needed
	createDir(machinePath())

	// copy ca.pem, cert.pem and key.pem into server dir
	for _, file := range []string{"ca.pem", "cert.pem", "key.pem"} {
		err := copyFile(certsFile(file), machineFile(file))
		if err != nil {
			fmt.Println("Fail to copy file from client to server path", err)
			os.Exit(1)
		}
	}

	// copy ssh key into server dir
	err = copyFile(*sshKeyPath, machineFile("id_rsa"))
	if err != nil {
		fmt.Println("Fail to copy ssh key")
		os.Exit(1)
	}

	// create server certs
	err = CreateServerCert(*serverIP, *serverDNS)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Server cert files sucessfully created")

	// create a config.json file as needed by docker-machine
	err = CreateConfigJSON()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("config.json sucessfully created")
}
