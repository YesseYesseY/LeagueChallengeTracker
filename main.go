package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
)

type JsDataChamps struct {
	Name string   `json:"name"`
	Key  string   `json:"key"`
	Tags []string `json:"tags"`
}

type JsDataCategories struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Simple      string `json:"simple"`
	Completed   []int  `json:"completed"`
}

type JsData struct {
	Champs     []JsDataChamps     `json:"champs"`
	Categories []JsDataCategories `json:"categories"`
}

func GetLockfileData() (string, string, error) {
	path := "C:\\Riot Games\\League of Legends\\lockfile"
	data_raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	data := string(data_raw)
	split := strings.Split(data, ":")
	return split[2], split[3], nil
}

func LcuGet(url string) (string, error) {
	port, password, err := GetLockfileData()

	if err != nil {
		return "", err
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	if url[0] != '/' {
		url = "/" + url
	}
	req, err := http.NewRequest("GET", "https://127.0.0.1:"+port+url, nil)

	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("riot:"+password)))

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	bodyraw, err := io.ReadAll(res.Body)

	if err != nil {
		return "", err
	}
	return string(bodyraw), nil
}

func GetData() (string, error) {
	var yes = &JsData{
		Champs:     []JsDataChamps{},
		Categories: []JsDataCategories{},
	}

	res, err := http.Get("https://ddragon.leagueoflegends.com/api/versions.json")

	if err != nil {
		return "", err
	}
	bodyraw, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var vers []string
	err = json.Unmarshal(bodyraw, &vers)
	latest_ver := vers[0]
	res, err = http.Get("https://ddragon.leagueoflegends.com/cdn/" + latest_ver + "/data/en_US/champion.json")

	if err != nil {
		return "", err
	}

	var chmps map[string]interface{}

	bodyraw, err = io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(bodyraw, &chmps)

	data := chmps["data"].(map[string]interface{})
	for _, v := range data {
		champ_data := v.(map[string]interface{})

		yes.Champs = append(yes.Champs, JsDataChamps{
			Name: champ_data["name"].(string),
			Key:  champ_data["key"].(string),
		})
	}

	sort.Slice(yes.Champs, func(i, j int) bool {
		if sort.StringsAreSorted([]string{
			yes.Champs[i].Name, yes.Champs[j].Name,
		}) {
			return true
		}
		return false
	})

	ids_to_check := map[string]string{
		"101301": "ARAM S-",
		"210001": "SR S+",
		"210002": "SR Penta",
		"202303": "SR No death",
		"120002": "CoopAI win",
		"401106": "SR Win",
		"401107": "Mastery 10",
		"401104": "Mastery 5",
		"602002": "Arena #1",
		"602001": "Arena played",
	}

	chall_raw, err := LcuGet("/lol-challenges/v1/challenges/local-player")
	if err != nil {
		return "", err
	}
	var chall map[string]interface{}
	err = json.Unmarshal([]byte(chall_raw), &chall)
	if err != nil {
		return "", err
	}

	keys := make([]string, 0, len(ids_to_check))
	for k := range ids_to_check {
		keys = append(keys, k)
	}

	for i := range keys {
		challenge := chall[keys[i]].(map[string]interface{})
		var to_add JsDataCategories = JsDataCategories{
			Name:        challenge["name"].(string),
			Description: challenge["description"].(string),
			Simple:      ids_to_check[keys[i]],
			Completed:   []int{},
		}
		if challenge["completedIds"] != nil {
			ids := challenge["completedIds"].([]interface{})
			for i := range ids {
				to_add.Completed = append(to_add.Completed, (int)(math.Floor(ids[i].(float64))))
			}
		}
		yes.Categories = append(yes.Categories, to_add)
	}

	sort.Slice(yes.Categories, func(i, j int) bool {
		if sort.StringsAreSorted([]string{
			yes.Categories[i].Name,
			yes.Categories[j].Name,
		}) {
			return true
		}
		return false
	})

	yes2, err := json.Marshal(yes)
	if err != nil {
		return "", err
	}

	return string(yes2), nil
}

func GetErrorPage(err error) string {
	result, err2 := os.ReadFile("./error.html")
	if err2 != nil {
		panic(err)
	}
	html_str := string(result)
	return strings.Replace(html_str, "<!--INSERT ERROR MSG HERE-->", fmt.Sprint(err), 1)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Getting data")
		data, err := GetData()
		if err != nil {
			w.Write([]byte(GetErrorPage(err)))
			return
		}

		result, err := os.ReadFile("./main.html")
		if err != nil {
			w.Write([]byte(GetErrorPage(err)))
			return
		}
		html_str := string(result)

		w.Write([]byte(strings.Replace(html_str, "/*REPLACE THIS WITH DATA*/", data, 1)))
	})

	fmt.Println("Server open, http://127.0.0.1:8090")
	http.ListenAndServe(":8090", nil)
}
