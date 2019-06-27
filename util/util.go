package util

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/btcsuite/fastsha256"
)

const APP_NAME string = "vertcoin-ocm"

func DataDirectory() string {
	if runtime.GOOS == "windows" {
		return path.Join(os.Getenv("APPDATA"), APP_NAME)
	} else if runtime.GOOS == "darwin" {
		return path.Join(os.Getenv("HOME"), "Library", "Application Support", APP_NAME)
	} else if runtime.GOOS == "linux" {
		return path.Join(os.Getenv("HOME"), fmt.Sprintf(".%s", strings.ToLower(APP_NAME)))
	}
	return "."
}

func ReplaceInFile(file string, find string, replace string) error {
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	output := bytes.Replace(input, []byte(find), []byte(replace), -1)

	if err = ioutil.WriteFile(file, output, 0666); err != nil {
		return err
	}

	return nil
}

type DifficultyResponse struct {
	Difficulty uint64 `json:"difficulty"`
}

func GetDifficulty() uint64 {
	diff := DifficultyResponse{}
	GetJson("https://insight.vertcoin.org/insight-vtc-api/status?q=getDifficulty", &diff)
	return diff.Difficulty
}

func GetNetHash() uint64 {
	difficulty := big.NewInt(int64(GetDifficulty()))
	netHash := difficulty.Mul(difficulty, big.NewInt(0).Exp(big.NewInt(2), big.NewInt(48), nil))
	return netHash.Div(netHash, big.NewInt(9830250)).Uint64() // 0xffff * blocktime in seconds
}

var jsonClient = &http.Client{Timeout: 10 * time.Second}

func GetJson(url string, target interface{}) error {
	r, err := jsonClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func UnpackZip(archive, unpackPath string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := path.Join(unpackPath, f.Name)
		os.MkdirAll(path.Dir(targetPath), 0700)
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}

	return nil
}

func UnpackTar(archive, unpackPath string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzf)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			targetPath := path.Join(unpackPath, name)
			os.MkdirAll(path.Dir(targetPath), 0700)
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ShaSum(file string) ([]byte, error) {
	h := fastsha256.New()
	fp, err := os.Open(file)
	if err != nil {
		return []byte{}, err
	}
	defer fp.Close()
	buf := make([]byte, 4096)

	for {
		n, err := fp.Read(buf)

		if err != nil && err != io.EOF {
			return []byte{}, err
		}

		if err == io.EOF {
			break
		} else {
			h.Write(buf[:n])
		}
	}
	return h.Sum(nil), nil
}
