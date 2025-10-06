package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"encoding/xml"

	"github.com/manifoldco/promptui"
	"github.com/sqweek/dialog"
)

var baseURL = "http://localhost:8080"

// --- Requêtes HTTP ---
func doRequest(method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := &http.Client{}
	return client.Do(req)
}

// --- Parser tous les <Name> dans le XML Gin ---
func parseAllNames(data []byte) []string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var names []string
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "Name" {
				var content string
				decoder.DecodeElement(&content, &t)
				names = append(names, content)
			}
		}
	}
	return names
}

// --- Parser erreur S3-like ---
func parseS3Error(data []byte) string {
	str := string(data)
	lines := splitOnce(str, "<Code>")
	if len(lines) < 2 {
		return str
	}
	code := splitOnce(lines[1], "</Code>")[0]
	lines2 := splitOnce(str, "<Message>")
	msg := ""
	if len(lines2) >= 2 {
		msg = splitOnce(lines2[1], "</Message>")[0]
	}
	return fmt.Sprintf("%s: %s", code, msg)
}

func splitOnce(s, sep string) []string {
	i := bytes.Index([]byte(s), []byte(sep))
	if i < 0 {
		return []string{s}
	}
	return []string{s[:i], s[i+len(sep):]}
}

// --- CLI principal ---
func main() {
	for {
		menu := []string{
			"Créer un bucket",
			"Lister les buckets",
			"Uploader un fichier",
			"Télécharger un objet",
			"Supprimer un objet",
			"Supprimer un bucket",
			"Quitter",
		}

		fmt.Println("======================================")
		prompt := promptui.Select{
			Label: "Choisissez une opération",
			Items: menu,
		}
		_, choice, err := prompt.Run()
		if err != nil {
			fmt.Println("Annulé")
			return
		}

		switch choice {
		case "Créer un bucket":
			prompt := promptui.Prompt{Label: "Nom du bucket"}
			bucket, _ := prompt.Run()
			resp, _ := doRequest(http.MethodPut, fmt.Sprintf("%s/%s", baseURL, bucket), nil, "")
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				fmt.Println("Erreur:", parseS3Error(data))
			} else {
				fmt.Println("Bucket créé:", bucket)
			}

		case "Lister les buckets":
			resp, _ := doRequest(http.MethodGet, baseURL+"/", nil, "")
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			buckets := parseAllNames(data)
			if len(buckets) == 0 {
				fmt.Println("Aucun bucket disponible")
			} else {
				fmt.Println("Buckets disponibles:")
				for _, b := range buckets {
					fmt.Println(" -", b)
				}
			}

		case "Uploader un fichier":
			resp, _ := doRequest(http.MethodGet, baseURL+"/", nil, "")
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			buckets := parseAllNames(data)
			if len(buckets) == 0 {
				fmt.Println("Aucun bucket existant, créez-en un d'abord.")
				break
			}

			prompt := promptui.Select{
				Label: "Sélectionner un bucket",
				Items: buckets,
			}
			_, selectedBucket, _ := prompt.Run()

			filePath, err := dialog.File().Title("Choisir un fichier à uploader").Load()
			if err != nil {
				fmt.Println("Aucun fichier sélectionné")
				break
			}
			objectName := filepath.Base(filePath)
			fileData, _ := os.ReadFile(filePath)
			resp2, _ := doRequest(http.MethodPut, fmt.Sprintf("%s/%s/%s", baseURL, selectedBucket, objectName),
				bytes.NewReader(fileData), "application/octet-stream")
			data2, _ := io.ReadAll(resp2.Body)
			resp2.Body.Close()
			if resp2.StatusCode >= 400 {
				fmt.Println("Erreur:", parseS3Error(data2))
			} else {
				fmt.Println("Fichier uploadé:", objectName, "dans bucket", selectedBucket)
			}

		case "Télécharger un objet":
			resp, _ := doRequest(http.MethodGet, baseURL+"/", nil, "")
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			buckets := parseAllNames(data)
			if len(buckets) == 0 {
				fmt.Println("Aucun bucket existant")
				break
			}
			prompt := promptui.Select{
				Label: "Sélectionner un bucket",
				Items: buckets,
			}
			_, selectedBucket, _ := prompt.Run()

			respObj, _ := doRequest(http.MethodGet, fmt.Sprintf("%s/%s", baseURL, selectedBucket), nil, "")
			objData, _ := io.ReadAll(respObj.Body)
			respObj.Body.Close()
			objectNames := parseAllNames(objData)
			if len(objectNames) == 0 {
				fmt.Println("Aucun objet dans ce bucket")
				break
			}
			promptObj := promptui.Select{
				Label: "Sélectionner un objet",
				Items: objectNames,
			}
			_, selectedObject, _ := promptObj.Run()

			respDownload, _ := doRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", baseURL, selectedBucket, selectedObject), nil, "")
			content, _ := io.ReadAll(respDownload.Body)
			respDownload.Body.Close()
			if respDownload.StatusCode >= 400 {
				fmt.Println("Erreur:", parseS3Error(content))
			} else {
				os.WriteFile(selectedObject, content, 0644)
				fmt.Println("Objet téléchargé:", selectedObject)
			}

		case "Supprimer un objet":
			resp, _ := doRequest(http.MethodGet, baseURL+"/", nil, "")
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			buckets := parseAllNames(data)
			if len(buckets) == 0 {
				fmt.Println("Aucun bucket existant")
				break
			}
			prompt := promptui.Select{
				Label: "Sélectionner un bucket",
				Items: buckets,
			}
			_, selectedBucket, _ := prompt.Run()

			respObj, _ := doRequest(http.MethodGet, fmt.Sprintf("%s/%s", baseURL, selectedBucket), nil, "")
			objData, _ := io.ReadAll(respObj.Body)
			respObj.Body.Close()
			objectNames := parseAllNames(objData)
			if len(objectNames) == 0 {
				fmt.Println("Aucun objet dans ce bucket")
				break
			}
			promptObj := promptui.Select{
				Label: "Sélectionner un objet à supprimer",
				Items: objectNames,
			}
			_, selectedObject, _ := promptObj.Run()

			respDel, _ := doRequest(http.MethodDelete, fmt.Sprintf("%s/%s/%s", baseURL, selectedBucket, selectedObject), nil, "")
			data2, _ := io.ReadAll(respDel.Body)
			respDel.Body.Close()
			if respDel.StatusCode >= 400 {
				fmt.Println("Erreur:", parseS3Error(data2))
			} else {
				fmt.Println("Objet supprimé:", selectedObject)
			}

		case "Supprimer un bucket":
			resp, _ := doRequest(http.MethodGet, baseURL+"/", nil, "")
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			buckets := parseAllNames(data)
			if len(buckets) == 0 {
				fmt.Println("Aucun bucket existant")
				break
			}
			prompt := promptui.Select{
				Label: "Sélectionner un bucket à supprimer",
				Items: buckets,
			}
			_, selectedBucket, _ := prompt.Run()

			respDel, _ := doRequest(http.MethodDelete, fmt.Sprintf("%s/%s", baseURL, selectedBucket), nil, "")
			data2, _ := io.ReadAll(respDel.Body)
			respDel.Body.Close()
			if respDel.StatusCode >= 400 {
				fmt.Println("Erreur:", parseS3Error(data2))
			} else {
				fmt.Println("Bucket supprimé:", selectedBucket)
			}

		case "Quitter":
			fmt.Println("Bye !")
			return
		}
	}
}
