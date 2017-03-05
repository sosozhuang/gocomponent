package module

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/containerops/configure"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/sosozhuang/component/types"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

var checkImageUrl string
var buildImageUrl string

func init() {
	checkImageUrl = configure.GetString("service.checkImage")
	if checkImageUrl == "" {
		log.Fatalln("Config file should specify service.checkImage")
		return
	}
	_, err := url.Parse(checkImageUrl)
	if err != nil {
		log.Fatalln("Configuration service.checkImage parse error:", err)
		return
	}

	buildImageUrl = configure.GetString("service.buildImage")
	if buildImageUrl == "" {
		log.Fatalln("Config file should specify service.buildImage")
		return
	}
	_, err = url.Parse(buildImageUrl)
	if err != nil {
		log.Fatalln("Configuration service.buildImage parse error:", err)
		return
	}

}

func CheckImageScript(req types.CheckImageScriptReq) error {
	if req.Script == "" {
		return errors.New("script content is empty")
	}
	body, err := json.Marshal(req)
	if err != nil {
		log.Errorln("CheckImageScript marshal request error:", err.Error())
		return errors.New("marshal data error: " + err.Error())
	}
	resp, err := http.Post(checkImageUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Errorf("CheckImageScript send reqeust to %s error: %s\n", checkImageUrl, err)
		return errors.New("send request error: " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Errorf("CheckImageScript send request to %s status code: %d\n", checkImageUrl, resp.StatusCode)
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorln("CheckImageScript read response body error:", err)
			return fmt.Errorf("response code: %d", resp.StatusCode)
		}
		var commonResp types.CommonResp
		err = json.Unmarshal(body, &commonResp)
		if err != nil {
			log.Errorln("CheckImageScript unmarshal response body error:", err)
			return fmt.Errorf("response code: %d", resp.StatusCode)
		}
		return fmt.Errorf("response code: %d, message: %s", resp.StatusCode, commonResp.Message)
	}
	return nil
}

// validateContextDirectory checks if all the contents of the directory
// can be read and returns an error if some files can't be read.
// Symlinks which point to non-existing files don't trigger an error
func validateContextDirectory(srcPath string, excludes []string) error {
	return filepath.Walk(filepath.Join(srcPath, "."), func(filePath string, f os.FileInfo, err error) error {
		// skip this directory/file if it's not in the path, it won't get added to the context
		if relFilePath, relErr := filepath.Rel(srcPath, filePath); relErr != nil {
			return relErr
		} else if skip, matchErr := fileutils.Matches(relFilePath, excludes); matchErr != nil {
			return matchErr
		} else if skip {
			if f.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// skip checking if symlinks point to non-existing files, such symlinks can be useful
		// also skip named pipes, because they hanging on open
		if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
			return nil
		}

		if !f.IsDir() {
			currentFile, err := os.Open(filePath)
			if err != nil && os.IsPermission(err) {
				return fmt.Errorf("no permission to read from '%s'", filePath)
			}
			currentFile.Close()
		}
		return nil
	})
}

func createTarStream(imageSetting types.ImageSetting) (io.ReadCloser, error) {
	dir, err := ioutil.TempDir("", "build-image-")
	if err != nil {
		return nil, err
	}
	defer os.Remove(dir)
	log.Debugf("CreateTarStream temp directory Created:", dir)
	if err := writeToFile(bytes.NewReader([]byte(imageSetting.Dockerfile)), dir, "Dockerfile", 0600); err != nil {
		return nil, err
	}
	if err := writeToFile(bytes.NewReader([]byte(imageSetting.ComponentStart)), dir, "component_start", 0700); err != nil {
		return nil, err
	}
	if err := writeToFile(bytes.NewReader([]byte(imageSetting.ComponentResult)), dir, "component_result", 0700); err != nil {
		return nil, err
	}
	if err := writeToFile(bytes.NewReader([]byte(imageSetting.ComponentStop)), dir, "component_stop", 0700); err != nil {
		return nil, err
	}

	includes := []string{"."}
	excludes := []string{}

	// If .dockerignore mentions .dockerignore or the Dockerfile
	// then make sure we send both files over to the daemon
	// because Dockerfile is, obviously, needed no matter what, and
	// .dockerignore is needed to know if either one needs to be
	// removed.  The deamon will remove them for us, if needed, after it
	// parses the Dockerfile.
	//
	// https://github.com/docker/docker/issues/8330
	//
	forceIncludeFiles := []string{filepath.Join(dir, "Dockerfile")}

	for _, includeFile := range forceIncludeFiles {
		if includeFile == "" {
			continue
		}
		keepThem, err := fileutils.Matches(includeFile, excludes)
		if err != nil {
			return nil, fmt.Errorf("cannot match .dockerfile: '%s', error: %s", includeFile, err)
		}
		if keepThem {
			includes = append(includes, includeFile)
		}
	}

	if err := validateContextDirectory(dir, excludes); err != nil {
		return nil, err
	}
	tarOpts := &archive.TarOptions{
		ExcludePatterns: excludes,
		IncludeFiles:    includes,
		Compression:     archive.Gzip,
		NoLchown:        true,
	}
	return archive.TarWithOptions(dir, tarOpts)
}

func writeToFile(src io.Reader, dir, fileName string, perm os.FileMode) error {
	name := filepath.Join(dir, fileName)
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, src); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if _, err := f.Stat(); err != nil {
		return err
	}
	return nil
}

func BuildImage(imageSetting types.ImageSetting) (*types.ImageInfo, error) {
	t, err := createTarStream(imageSetting)
	if err != nil {
		return nil, errors.New("create tar stream error: " + err.Error())
	}
	defer t.Close()
	data, err := ioutil.ReadAll(t)
	if err != nil {
		return nil, errors.New("read tar stream error: " + err.Error())
	}
	var req types.BuildImageReq
	req.TarStream = data
	req.ImageInfo = imageSetting.ImageInfo
	req.PushInfo = imageSetting.PushInfo
	body, err := json.Marshal(req)
	if err != nil {
		log.Errorln("BuildImage marshal request error:", err.Error())
		return nil, errors.New("marshal data error: " + err.Error())
	}
	resp, err := http.Post(buildImageUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Errorf("BuildImage send reqeust to %s error: %s\n", buildImageUrl, err)
		return nil, errors.New("send request error: " + err.Error())
	}
	defer resp.Body.Close()
	var buildImageResp types.BuildImageResp
	if resp.StatusCode != http.StatusOK {
		log.Errorf("BuildImage send request to %s status code: %d\n", buildImageUrl, resp.StatusCode)
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorln("BuildImage read response body error:", err)
			return nil, fmt.Errorf("response code: %d", resp.StatusCode)
		}
		err = json.Unmarshal(body, &buildImageResp)
		if err != nil {
			log.Errorln("BuildImage unmarshal response body error:", err)
			return nil, fmt.Errorf("response code: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("response code: %d, message: %s", resp.StatusCode, buildImageResp.Message)
	}
	return buildImageResp.ImageInfo, nil
}
