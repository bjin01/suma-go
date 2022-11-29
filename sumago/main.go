package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Sumaconf struct {
	Server        string `json:"-" yaml:"server"`
	User          string `json:"login" yaml:"user"`
	Password      string `json:"password" yaml:"password"`
	cookie_key    string
	cookie_val    string
	cookie_maxAge int
}

var sumaconf Sumaconf
var jobstart *string

type ListActiveSystem struct {
	Success bool `json:"success"`
	Result  []struct {
		LastBoot    string `json:"last_boot"`
		Name        string `json:"name"`
		ID          int    `json:"id"`
		LastCheckin string `json:"last_checkin"`
		Packages    ListLatestUpgradablePackages
		JobIDs      []UpdateJob
	} `json:"result"`
}

type ListLatestUpgradablePackages struct {
	Success bool `json:"success"`
	Result  []struct {
		FromEpoch   string `json:"from_epoch"`
		ToRelease   string `json:"to_release"`
		Name        string `json:"name"`
		FromRelease string `json:"from_release"`
		ToEpoch     string `json:"to_epoch"`
		Arch        string `json:"arch"`
		ToPackageID int    `json:"to_package_id"`
		FromVersion string `json:"from_version"`
		ToVersion   string `json:"to_version"`
		FromArch    string `json:"from_arch"`
		ToArch      string `json:"to_arch"`
	} `json:"result"`
}

type SystemScheduleUpdate struct {
	Sid                int    `json:"sid"`
	PackageIds         []int  `json:"packageIds"`
	EarliestOccurrence string `json:"earliestOccurrence"`
}

type UpdateJob struct {
	Success bool `json:"success"`
	JobID   int  `json:"result"`
}

func myCookieJar() *cookiejar.Jar {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Got error while creating cookie jar %s", err.Error())
	}
	return jar
}

func httpTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.TLSClientConfig.InsecureSkipVerify = true
	return t
}

func init() {

	var conf_file = flag.String("sumaconf", "", "provide the suma conf file with login data.")
	jobstart = flag.String("schedule", "", "provide number of hours to start the job from now.")
	flag.Parse()

	if len(*conf_file) == 0 {
		log.Fatal("sumaconf not provided. Exit")
	}

	yfile, err := ioutil.ReadFile(*conf_file)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yfile, &sumaconf)
	if err != nil {
		log.Fatal(err)
	}
}

func httpClient() *http.Client {

	transport := http.Transport{
		/* Proxy: func(*http.Request) (*url.URL, error) {
		},
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
		},
		Dial: func(network string, addr string) (net.Conn, error) {
		},
		DialTLSContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
		},
		DialTLS: func(network string, addr string) (net.Conn, error) {
		},
		TLSClientConfig:       &tls.Config{}, */
		TLSHandshakeTimeout:   0,
		DisableKeepAlives:     true,
		DisableCompression:    false,
		MaxIdleConns:          0,
		MaxIdleConnsPerHost:   0,
		MaxConnsPerHost:       0,
		IdleConnTimeout:       0,
		ResponseHeaderTimeout: 0,
		ExpectContinueTimeout: 0,
		/* TLSNextProto:          map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		ProxyConnectHeader:    map[string][]string{},
		GetProxyConnectHeader: func(ctx context.Context, proxyURL *url.URL, target string) (http.Header, error) {
		}, */
		MaxResponseHeaderBytes: 0,
		WriteBufferSize:        0,
		ReadBufferSize:         0,
		ForceAttemptHTTP2:      false,
	}

	client := &http.Client{
		Transport: &transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar:     myCookieJar(),
		Timeout: 15 * time.Second,
	}
	return client
}

func (l *Sumaconf) Loginsuma(client *http.Client) error {

	e, err := json.Marshal(l)
	if err != nil {
		fmt.Println(err)
		return err
	}

	login_url := fmt.Sprintf("https://%s/rhn/manager/api/auth/login", l.Server)
	req, err := http.NewRequest("POST", login_url, bytes.NewBuffer(e))
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured. Error is: %s", err.Error())
	}
	defer resp.Body.Close()

	if len(resp.Cookies()) != 0 {
		urlObj, _ := url.Parse(fmt.Sprintf("https://%s", l.Server))

		for a, c := range resp.Cookies() {
			fmt.Printf("cookie key: %s; val: %s, valid: %#v\n", c.Name, c.Value, c.Valid())
			if c.Name == "pxt-session-cookie" && c.MaxAge >= 30 {
				client.Jar.SetCookies(urlObj, []*http.Cookie{resp.Cookies()[a]})
				l.cookie_key = c.Name
				l.cookie_val = c.Value
			}
		}

	}

	if resp.StatusCode == 200 {
		fmt.Println("Login successful.")
	} else {
		fmt.Println("Login failed.")
	}

	return nil
}

func (l *Sumaconf) CreateRequest(request_type string, url string, e []byte) (*http.Request, error) {
	req, err := http.NewRequest(request_type, "", nil)
	if err != nil {
		return nil, fmt.Errorf("Got error while creating request object %s\n", err.Error())
	}
	if e != nil {
		req, err = http.NewRequest(request_type, url, bytes.NewBuffer(e))
		if err != nil {
			return nil, fmt.Errorf("Got error while creating request object %s\n", err.Error())
		}
	} else {
		req, err = http.NewRequest(request_type, url, nil)
		if err != nil {
			return nil, fmt.Errorf("Got error while creating request object %s\n", err.Error())
		}
	}

	/* req.AddCookie(&http.Cookie{
		Name:       l.cookie_key,
		Value:      l.cookie_val,
		Path:       "",
		Domain:     "",
		Expires:    time.Time{},
		RawExpires: "",
		MaxAge:     0,
		Secure:     false,
		HttpOnly:   false,
		SameSite:   0,
		Raw:        "",
		Unparsed:   []string{},
	}) */
	return req, nil
}

func (l *ListActiveSystem) Getsystems(client *http.Client, sumaconf *Sumaconf) error {

	err := sumaconf.Loginsuma(client)
	if err != nil {
		log.Fatalf("sumaconf login got error %s", err.Error())
	}

	url, _ := url.Parse(fmt.Sprintf("https://%s/rhn/manager/api/system/listActiveSystems", sumaconf.Server))
	req, err := sumaconf.CreateRequest("GET", url.String(), nil)
	fmt.Printf("cookie in req: %#v\n", fmt.Sprintf("%s", req.Cookies()))
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
	}

	//fmt.Printf("listactivesystem statuscode %d\n", resp.StatusCode)
	//fmt.Println(resp.Request.URL)

	x := 0
	for resp.StatusCode == 401 && x < 5 {
		fmt.Println("Oops, listactivesystems returns 401, retry in 2 seconds...")
		time.Sleep(time.Second * 2)
		x++
		resp, err = client.Do(req)
		if err != nil {
			log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
		}
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Printf("full resp.body: %+v\n", string(body))
	json.Unmarshal([]byte(body), l)
	//fmt.Printf("lets see listactivesystem: %+v\n", l)

	if l.Success != true || len(l.Result) == 0 {
		return errors.New(fmt.Sprintf("API call %s failed or no active systems found.", url))

	}
	return nil
}

func (u *ListActiveSystem) Getpackages(client *http.Client, sumaconf *Sumaconf) error {
	url, _ := url.Parse(fmt.Sprintf("https://%s/rhn/manager/api/system/listLatestUpgradablePackages", sumaconf.Server))

	fmt.Printf("Active Systems: \n")
	for i, a := range u.Result {
		listupgpkgs := new(ListLatestUpgradablePackages)
		req, err := sumaconf.CreateRequest("GET", url.String(), nil)
		if err != nil {
			log.Fatalf("Got error %s", err.Error())
		}
		fmt.Printf("%s\n\tID: %d\n\tLast Boot: %s\n", a.Name, a.ID, a.LastBoot)
		q := req.URL.Query()
		q.Add("sid", fmt.Sprintf("%d", a.ID))

		req.URL.RawQuery = q.Encode()
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
		}

		x := 0
		for resp.StatusCode == 401 && x < 5 {
			fmt.Println("Oops, Getpackages returns 401, retry in 2 seconds...")
			time.Sleep(time.Second * 2)
			x++
			resp, err = client.Do(req)
			if err != nil {
				log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
			}
		}

		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		//fmt.Printf("%s", string(body))
		//var p listLatestUpgradablePackages
		json.Unmarshal([]byte(body), listupgpkgs)

		//fmt.Printf("packages %+v", u)
		fmt.Printf("\tUpgradable packages %d\n", len(listupgpkgs.Result))
		//a.Packages = len(u.Result)
		u.Result[i].Packages = *listupgpkgs

	}

	return nil
}

func getISOtime(scheduleTime string) string {
	t := time.Now()
	scheduleTime = scheduleTime + "h"
	s, _ := time.ParseDuration(scheduleTime)
	AfteroneHour := t.Add(s)
	return AfteroneHour.Format(time.RFC3339)
}

func (u *ListActiveSystem) InstallUpdates(client *http.Client, sumaconf *Sumaconf, scheduleTime string) error {
	url, _ := url.Parse(fmt.Sprintf("https://%s/rhn/manager/api/system/schedulePackageInstall", sumaconf.Server))
	for i, a := range u.Result {
		systemToupdate := new(SystemScheduleUpdate)
		//systemToupdate.SessionKey = sumaconf.cookie_val
		systemToupdate.Sid = a.ID
		if len(a.Packages.Result) > 0 {
			for _, p := range a.Packages.Result {
				systemToupdate.PackageIds = append(systemToupdate.PackageIds, p.ToPackageID)
			}
		} else {
			fmt.Printf("Skip system: %s, no updates to install.\n", a.Name)
			continue
		}
		if len(systemToupdate.PackageIds) > 0 && strings.TrimSpace(scheduleTime) != "" {
			systemToupdate.EarliestOccurrence = getISOtime(scheduleTime)
		} else {
			fmt.Printf("Skip system: %s, no time schedule given.\n", a.Name)
			break
		}

		//fmt.Printf("%#v", systemToupdate)
		e, err := json.Marshal(systemToupdate)
		//fmt.Printf("\nmarshal systemToupdate: %s\n", fmt.Sprintf("%s", e))
		req, err := sumaconf.CreateRequest("POST", url.String(), e)
		if err != nil {
			log.Fatalf("Got error %s", err.Error())
		}
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error occured. Error is: %s", err.Error())
		}

		for resp.StatusCode == 401 {
			fmt.Println("Oops, schedule update job returns 401, retry retry in 2 seconds...")
			time.Sleep(time.Second * 2)
			resp, err = client.Do(req)
			if err != nil {
				log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
			}
		}

		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		//fmt.Printf("result body: %#v", fmt.Sprintf("%s", body))
		jobresult := new(UpdateJob)
		json.Unmarshal([]byte(body), jobresult)
		//fmt.Printf("lets see listactivesystem: %+v", &listactivesystems)

		if jobresult.Success != true || jobresult.JobID == 0 {
			return errors.New(fmt.Sprintf("API call %s failed, no schedule update job created. %#v", url, jobresult))

		} else {
			fmt.Printf("\nJob: %d created for %s\n", jobresult.JobID, a.Name)
			u.Result[i].JobIDs = append(u.Result[i].JobIDs, *jobresult)
		}
	}
	return nil
}

func (l *Sumaconf) sumalogout(client *http.Client) error {
	url := fmt.Sprintf("https://%s/rhn/manager/api/auth/logout", l.Server)
	req, err := l.CreateRequest("POST", url, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
	}

	x := 0
	for resp.StatusCode == 401 && x < 5 {
		fmt.Println("Oops, logout returns 401, retry retry in 2 seconds...")
		time.Sleep(time.Second * 2)
		x++
		resp, err = client.Do(req)
		if err != nil {
			log.Fatalf("Error occured while calling %s Error is: %s", url, err.Error())
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Logout failed %d", resp.StatusCode)
	} else {
		fmt.Printf("SUMA logout.\n")
	}
	return nil
}

func main() {
	client := httpClient()
	listactivesystems := new(ListActiveSystem)
	err := listactivesystems.Getsystems(client, &sumaconf)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	//listupgradablepackages := new(ListLatestUpgradablePackages)
	err = listactivesystems.Getpackages(client, &sumaconf)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	/* err = listactivesystems.InstallUpdates(client, &sumaconf, *jobstart)
	if err != nil {
		log.Fatalf("%s", err.Error())
	} */

	//fmt.Printf("in main: no of upgradable packages: %+v\n", listactivesystems)
	err = sumaconf.sumalogout(client)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	/* for _, b := range listactivesystems.Result {
		fmt.Printf("%s: %d packages\n", b.Name, len(b.Packages.Result))
		for _, d := range b.Packages.Result {
			fmt.Printf("\t%s\n", d.Name)
		}
	} */

}
