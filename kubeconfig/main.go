package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"encoding/base64"
	"gopkg.in/yaml.v2"
)

type B64 []byte

func (b *B64) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = data
	return nil
}

func (b B64) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(b), nil
}

func dataOrFile(data B64, filename string) (B64, error) {
	if len(data) > 0 {
		return data, nil
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type ClusterInfo struct {
	CertificateAuthorityData B64    `yaml:"certificate-authority-data,omitempty"`
	CertificateAuthority     string `yaml:"certificate-authority,omitempty"`
	Server                   string `yaml:"server,omitempty"`
}

func (ci ClusterInfo) MarshalYAML() (interface{}, error) {
	b, err := dataOrFile(ci.CertificateAuthorityData, ci.CertificateAuthority)
	if err != nil {
		return nil, err
	}
	return struct {
		CertificateAuthorityData B64    `yaml:"certificate-authority-data,omitempty"`
		Server                   string `yaml:"server,omitempty"`
	}{
		CertificateAuthorityData: b,
		Server:                   ci.Server,
	}, nil
}

type Cluster struct {
	Name    string      `yaml:"name,omitempty"`
	Cluster ClusterInfo `yaml:"cluster,omitempty"`
}

type ContextInfo struct {
	Cluster string `yaml:"cluster,omitempty"`
	User    string `yaml:"user,omitempty"`
}

type Context struct {
	Name    string      `yaml:"name,omitempty"`
	Context ContextInfo `yaml:"context,omitempty"`
}

type UserInfo struct {
	ClientCertificateData B64    `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         B64    `yaml:"client-key-data,omitempty"`
	ClientCertificate     string `yaml:"client-certificate,omitempty"`
	ClientKey             string `yaml:"client-key,omitempty"`
}

func (ui UserInfo) MarshalYAML() (interface{}, error) {
	cert, err := dataOrFile(ui.ClientCertificateData, ui.ClientCertificate)
	if err != nil {
		return nil, err
	}
	key, err := dataOrFile(ui.ClientKeyData, ui.ClientKey)
	if err != nil {
		return nil, err
	}
	return struct {
		ClientCertificateData B64 `yaml:"client-certificate-data,omitempty"`
		ClientKeyData         B64 `yaml:"client-key-data,omitempty"`
	}{
		ClientCertificateData: cert,
		ClientKeyData:         key,
	}, nil
}

type User struct {
	Name string   `yaml:"name,omitempty"`
	User UserInfo `yaml:"user,omitempty"`
}

type Config struct {
	ApiVersion     string    `yaml:"apiVersion,omitempty"`
	Clusters       []Cluster `yaml:"clusters,omitempty"`
	Contexts       []Context `yaml:"contexts,omitempty"`
	CurrentContext string    `yaml:"current-context,omitempty"`
	Kind           string    `yaml:"kind,omitempty"`
	Users          []User    `yaml:"users,omitempty"`
	Preferences    struct{}  `yaml:"preferences,omitempty"`
}

func (c *Config) FindCluster(name string) *Cluster {
	for i := range c.Clusters {
		cluster := &c.Clusters[i]
		if cluster.Name == name {
			return cluster
		}
	}
	return nil
}

func (c *Config) FindContext(name string) *Context {
	for i := range c.Contexts {
		ctx := &c.Contexts[i]
		if ctx.Name == name {
			return ctx
		}
	}
	return nil
}

func (c *Config) FindUser(name string) *User {
	for i := range c.Users {
		user := &c.Users[i]
		if user.Name == name {
			return user
		}
	}
	return nil
}

func main() {
	var (
		fname   string
		context string
	)
	flag.StringVar(&fname, "f", "", "input kube config file name (~/.kube/*.conf)")
	flag.StringVar(&context, "c", "", "context name")
	flag.Parse()

	data, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Fatalf("unable to read from stdin: %v", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatalf("unable to load config: %v", err)
	}

	// find
	ctx := cfg.FindContext(context)
	if ctx == nil {
		log.Fatalf("unable to find context %q", context)
	}
	cfg.Contexts = []Context{*ctx}

	cluster := cfg.FindCluster(ctx.Context.Cluster)
	if cluster == nil {
		log.Fatalf("unable to find cluster %q", ctx.Context.Cluster)
	}
	cfg.Clusters = []Cluster{*cluster}

	user := cfg.FindUser(ctx.Context.User)
	if user == nil {
		log.Fatalf("unable to find user %q", ctx.Context.User)
	}
	cfg.Users = []User{*user}

	cfg.CurrentContext = context

	// output
	data, err = yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("unable to marshal config: %v", err)
	}

	_, err = io.Copy(os.Stdout, bytes.NewReader(data))
	if err != nil {
		log.Fatalf("unable to marshal config: %v", err)
	}
}
