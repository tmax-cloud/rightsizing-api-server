package options

import (
	"errors"
	"os"

	"github.com/akamensky/argparse"
)

type Options struct {
	LogFile  *string
	CertFile *string
	KeyFile  *string
	Mode     *string
	Port     *int
	GrpcHost *string
	GrpcPort *string
	parser   *argparse.Parser
}

func NewOptions() (*Options, error) {
	option := &Options{}

	parser := argparse.NewParser("print", "Argument Parser for api-server configurations")
	option.LogFile = parser.String("l", "log-file", &argparse.Options{
		Help:    "log-file name",
		Default: "/var/log/app.log",
	})
	option.CertFile = parser.String("", "tls-cert-file", &argparse.Options{
		Help: "CertFile containing the defaultx509 Certificate for HTTPS. (CA cert)",
	})
	option.KeyFile = parser.String("", "tls-private-key-file", &argparse.Options{
		Help: "Private key file containing the default x509 private key matching --tls-cert-file",
	})
	option.Port = parser.Int("p", "port", &argparse.Options{
		Help:    "The port used by api-server",
		Default: 8000,
	})
	option.Mode = parser.Selector("m", "mode", []string{"release", "development", "debug"}, &argparse.Options{
		Help:    "Choose release/development mode (default debug mode)",
		Default: "debug",
	})
	option.GrpcHost = parser.String("", "grpc-host", &argparse.Options{
		Help:    "The host for grpc client",
		Default: "localhost",
	})

	option.GrpcPort = parser.String("", "grpc-port", &argparse.Options{
		Help:    "The port used by grpc client",
		Default: "50051",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		return nil, err
	}

	if err := option.Validate(); err != nil {
		return nil, err
	}

	option.parser = parser
	return option, nil
}

func (o *Options) Validate() error {
	if (o.CertFile == nil && o.KeyFile != nil) || (o.CertFile != nil && o.KeyFile == nil) {
		return errors.New("certificate/private key both must be present or neither must be present")
	}

	if *o.GrpcHost == "" || *o.GrpcPort == "" {
		return errors.New("grpc host and port both must be present")
	}
	return nil
}

func (o *Options) Usage(err error) string {
	return o.parser.Usage(err)
}
