package main

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"tyk/tyk/bootstrap/data"
	"tyk/tyk/bootstrap/helpers"
	"tyk/tyk/bootstrap/readiness"
	"tyk/tyk/bootstrap/tyk"
)

func main() {
	conf, err := data.NewConfig()
	if err != nil {
		exit(err)
	}

	logger := logrus.New()

	err = readiness.CheckIfRequiredDeploymentsAreReady()
	if err != nil {
		exit(err)
	}

	tykSvc := tyk.NewService(conf, logger)

	orgExists := false
	if err = tykSvc.OrgExists(); err != nil {
		if !errors.Is(err, tyk.ErrOrgExists) {
			exit(err)
		}

		orgExists = true
	}

	if !orgExists {
		if err = tykSvc.CreateOrganisation(); err != nil {
			exit(err)
		}
	}

	admin, err := tykSvc.UserExists(conf.Tyk.Admin.EmailAddress)
	if err != nil {
		exit(err)
	}

	if admin == nil {
		if err = tykSvc.CreateAdmin(); err != nil {
			exit(err)
		}
	}

	if conf.BootstrapPortal {
		if err = tykSvc.BootstrapClassicPortal(); err != nil {
			exit(err)
		}
	}

	if data.BootstrapConf.OperatorKubernetesSecretName != "" {
		err = helpers.BootstrapTykOperatorSecret()
		if err != nil {
			exit(err)
		}
	}

	fmt.Println("Finished bootstrapping operator secret\nStarted bootstrapping portal secret")

	if data.BootstrapConf.DevPortalKubernetesSecretName != "" {
		err = helpers.BootstrapTykPortalSecret()
		if err != nil {
			exit(err)
		}
	}

}

func exit(err error) {
	if err == nil {
		os.Exit(0)
	}

	fmt.Println(err)
	os.Exit(1)
}
