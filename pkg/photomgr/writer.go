package photomgr

import (
	"bytes"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/nfnt/resize"
)

type Manager struct {
	dir string
}

func (m *Manager) SaveImage(data []byte, user uint64, name string) error {
	jimg, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	log.Println("GOT IMG", name, jimg.Bounds().Size())

	newImage := resize.Resize(256, 256, jimg, resize.Lanczos3)

	buf := bytes.Buffer{}

	err = jpeg.Encode(&buf, newImage, nil)
	if err != nil {
		return err
	}

	path := m.dir + "/" + strconv.FormatUint(user, 10)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0777)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(path+"/"+name+".jpg", buf.Bytes(), 0666)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) GetImage(user uint64, name string) ([]byte, error) {
	path := m.dir + "/" + strconv.FormatUint(user, 10)

	bts, err := ioutil.ReadFile(path + "/" + name)
	if err != nil {
		return nil, err
	}

	return bts, nil
}

func (m *Manager) ListImages(user uint64) ([]string, error) {
	path := m.dir + "/" + strconv.FormatUint(user, 10)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	imgs := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".jpg") {
			imgs = append(imgs, f.Name())
		}
	}

	return imgs, nil
}

func New(dir string) *Manager {
	return &Manager{dir: dir}
}
