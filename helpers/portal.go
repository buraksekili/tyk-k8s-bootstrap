package helpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"tyk/tyk/bootstrap/data"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func BoostrapPortal(client http.Client) error {
	err := CreatePortalDefaultSettings(client)
	if err != nil {
		return err
	}

	err = InitialiseCatalogue(client)
	if err != nil {
		return err
	}

	err = CreatePortalHomepage(client)
	if err != nil {
		return err
	}

	err = SetPortalCname(client)
	if err != nil {
		return err
	}

	return nil
}

type InitCatalogReq struct {
	OrgId string `json:"org_id"`
}

type CnameRequest struct {
	Cname string `json:"cname"`
}

func SetPortalCname(client http.Client) error {
	fmt.Println("Setting portal cname")

	cnameReq := CnameRequest{Cname: data.BootstrapConf.Tyk.Org.Cname}

	reqBody, err := json.Marshal(cnameReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		data.BootstrapConf.K8s.DashboardSvcUrl+ApiPortalCnameEndpoint,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	req.Header.Set(data.AuthorizationHeader, data.BootstrapConf.Tyk.UserAuth)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("failed to set portal cname")
	}

	// restarting the dashboard to apply the new cname
	return RestartDashboard()
}

func InitialiseCatalogue(client http.Client) error {
	fmt.Println("Initialising Catalogue")

	initCatalog := InitCatalogReq{OrgId: data.BootstrapConf.Tyk.OrgId}

	reqBody, err := json.Marshal(initCatalog)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		data.BootstrapConf.K8s.DashboardSvcUrl+ApiPortalCatalogueEndpoint,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	req.Header.Set(data.AuthorizationHeader, data.BootstrapConf.Tyk.UserAuth)

	res, err := client.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		return err
	}

	resp := DashboardGeneralResponse{}

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

func CreatePortalHomepage(client http.Client) error {
	fmt.Println("Creating portal homepage")

	homepageContents := GetPortalHomepage()

	reqBody, err := json.Marshal(homepageContents)
	if err != nil {
		return err
	}

	reqData := bytes.NewReader(reqBody)

	req, err := http.NewRequest(http.MethodPost, data.BootstrapConf.K8s.DashboardSvcUrl+ApiPortalPagesEndpoint, reqData)
	if err != nil {
		return err
	}

	req.Header.Set(data.AuthorizationHeader, data.BootstrapConf.Tyk.UserAuth)

	res, err := client.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		return err
	}

	resp := DashboardGeneralResponse{}

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

func GetPortalHomepage() PortalHomepageRequest {
	return PortalHomepageRequest{
		IsHomepage:   true,
		TemplateName: "",
		Title:        "Developer portal name",
		Slug:         "/",
		Fields: PortalFields{
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

type PortalHomepageRequest struct {
	IsHomepage   bool         `json:"is_homepage"`
	TemplateName string       `json:"template_name"`
	Title        string       `json:"title"`
	Slug         string       `json:"slug"`
	Fields       PortalFields `json:"fields"`
}

type PortalFields struct {
	JumboCTATitle       string `json:"JumboCTATitle"`
	SubHeading          string `json:"SubHeading"`
	JumboCTALink        string `json:"JumboCTALink"`
	JumboCTALinkTitle   string `json:"JumboCTALinkTitle"`
	PanelOneContent     string `json:"PanelOneContent"`
	PanelOneLink        string `json:"PanelOneLink"`
	PanelOneLinkTitle   string `json:"PanelOneLinkTitle"`
	PanelOneTitle       string `json:"PanelOneTitle"`
	PanelThereeContent  string `json:"PanelThereeContent"`
	PanelThreeContent   string `json:"PanelThreeContent"`
	PanelThreeLink      string `json:"PanelThreeLink"`
	PanelThreeLinkTitle string `json:"PanelThreeLinkTitle"`
	PanelThreeTitle     string `json:"PanelThreeTitle"`
	PanelTwoContent     string `json:"PanelTwoContent"`
	PanelTwoLink        string `json:"PanelTwoLink"`
	PanelTwoLinkTitle   string `json:"PanelTwoLinkTitle"`
	PanelTwoTitle       string `json:"PanelTwoTitle"`
}

func CreatePortalDefaultSettings(client http.Client) error {
	fmt.Println("Creating bootstrap default settings")

	// TODO(buraksekili): DashboardSvcUrl can be populated via environment variables. So, the URL
	// might have trailing slashes. Constructing the URL with raw string concatenating is not a good
	// approach here. Needs refactoring.
	req, err := http.NewRequest(
		http.MethodPut,
		data.BootstrapConf.K8s.DashboardSvcUrl+ApiPortalConfigurationEndpoint,
		nil,
	)
	req.Header.Set(data.AuthorizationHeader, data.BootstrapConf.Tyk.UserAuth)

	if err != nil {
		return err
	}

	res, err := client.Do(req)
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
