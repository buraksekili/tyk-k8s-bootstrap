package tyk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"time"
	"tyk/tyk/bootstrap/data"
	"tyk/tyk/bootstrap/tyk/api"
	"tyk/tyk/bootstrap/tyk/internal/constants"
)

func (s *Service) BootstrapClassicPortal() error {
	err := s.createPortalDefaultSettings()
	if err != nil {
		return err
	}

	err = s.initialiseCatalogue()
	if err != nil {
		return err
	}

	err = s.createPortalHomePage()
	if err != nil {
		return err
	}

	err = s.setPortalCname()
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) setPortalCname() error {
	fmt.Println("Setting portal cname")

	cnameReq := api.CnameReq{Cname: s.appArgs.Tyk.Org.Cname}

	reqBody, err := json.Marshal(cnameReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		s.appArgs.K8s.DashboardSvcUrl+constants.ApiPortalCnameEndpoint,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	req.Header.Set(data.AuthorizationHeader, s.appArgs.Tyk.Admin.Auth)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("failed to set portal cname")
	}

	// restarting the dashboard to apply the new cname
	return RestartDashboard()
}

func (s *Service) initialiseCatalogue() error {
	fmt.Println("Initialising Catalogue")

	initCatalog := api.InitCatalogReq{OrgId: s.appArgs.Tyk.Org.ID}

	reqBody, err := json.Marshal(initCatalog)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.appArgs.K8s.DashboardSvcUrl+constants.ApiPortalCatalogueEndpoint,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	req.Header.Set(data.AuthorizationHeader, s.appArgs.Tyk.Admin.Auth)

	res, err := s.httpClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		return err
	}

	resp := api.DashboardAPIResp{}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(bodyBytes, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) createPortalHomePage() error {
	fmt.Println("Creating portal homepage")

	homepageContents := portalHomepageReq()

	reqBody, err := json.Marshal(homepageContents)
	if err != nil {
		return err
	}

	reqData := bytes.NewReader(reqBody)

	req, err := http.NewRequest(http.MethodPost, s.appArgs.K8s.DashboardSvcUrl+constants.ApiPortalPagesEndpoint, reqData)
	if err != nil {
		return err
	}

	req.Header.Set(data.AuthorizationHeader, s.appArgs.Tyk.Admin.Auth)

	res, err := s.httpClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		return err
	}

	resp := api.DashboardAPIResp{}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(bodyBytes, &resp)
	if err != nil {
		return err
	}

	return nil
}

func portalHomepageReq() api.PortalHomepageReq {
	return api.PortalHomepageReq{
		IsHomepage:   true,
		TemplateName: "",
		Title:        "Developer portal name",
		Slug:         "/",
		Fields: api.PortalFields{
			JumboCTATitle:       "Tyk Developer Portal",
			SubHeading:          "Sub Header",
			JumboCTALink:        "#cta",
			JumboCTALinkTitle:   "Your awesome APIs, hosted with Tyk!",
			PanelOneContent:     "Panel 1 content.",
			PanelOneLink:        "#panel1",
			PanelOneLinkTitle:   "Panel 1 Button",
			PanelOneTitle:       "Panel 1 Title",
			PanelThereeContent:  "",
			PanelThreeContent:   "Panel 3 content.",
			PanelThreeLink:      "#panel3",
			PanelThreeLinkTitle: "Panel 3 Button",
			PanelThreeTitle:     "Panel 3 Title",
			PanelTwoContent:     "Panel 2 content.",
			PanelTwoLink:        "#panel2",
			PanelTwoLinkTitle:   "Panel 2 Button",
			PanelTwoTitle:       "Panel 2 Title",
		},
	}
}

func (s *Service) createPortalDefaultSettings() error {
	fmt.Println("Creating bootstrap default settings")

	// TODO(buraksekili): DashboardSvcUrl can be populated via environment variables. So, the URL
	// might have trailing slashes. Constructing the URL with raw string concatenating is not a good
	// approach here. Needs refactoring.
	req, err := http.NewRequest(
		http.MethodPut,
		s.appArgs.K8s.DashboardSvcUrl+constants.ApiPortalConfigurationEndpoint,
		nil,
	)
	req.Header.Set(data.AuthorizationHeader, s.appArgs.Tyk.Admin.Auth)

	if err != nil {
		return err
	}

	res, err := s.httpClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		return err
	}

	return nil
}

func RestartDashboard() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	if data.BootstrapConf.K8s.DashboardDeploymentName == "" {
		ls := metav1.LabelSelector{MatchLabels: map[string]string{
			data.TykBootstrapLabel: data.TykBootstrapDashboardDeployLabel,
		}}

		deployments, err := clientset.
			AppsV1().
			Deployments(data.BootstrapConf.K8s.ReleaseNamespace).
			List(
				context.TODO(),
				metav1.ListOptions{
					LabelSelector: labels.Set(ls.MatchLabels).String(),
				},
			)
		if err != nil {
			return fmt.Errorf("failed to list Tyk Dashboard Deployment, err: %v", err)
		}

		for i := range deployments.Items {
			data.BootstrapConf.K8s.DashboardDeploymentName = deployments.Items[i].ObjectMeta.Name
		}
	}

	timeStamp := fmt.Sprintf(
		`{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`,
		time.Now().Format("20060102150405"),
	)

	_, err = clientset.
		AppsV1().
		Deployments(data.BootstrapConf.K8s.ReleaseNamespace).
		Patch(
			context.TODO(),
			data.BootstrapConf.K8s.DashboardDeploymentName,
			types.StrategicMergePatchType,
			[]byte(timeStamp),
			metav1.PatchOptions{},
		)

	return err
}
