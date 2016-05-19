package qemu

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/emc-advanced-dev/pkg/errors"
	unikos "github.com/emc-advanced-dev/unik/pkg/os"
	"github.com/emc-advanced-dev/unik/pkg/types"
)

func (p *QemuProvider) Stage(params types.StageImageParams) (_ *types.Image, err error) {
	images, err := p.ListImages()
	if err != nil {
		return nil, errors.New("retrieving image list for existing image", err)
	}

	for _, image := range images {
		if image.Name == params.Name {
			if !params.Force {
				return nil, errors.New("an image already exists with name '"+params.Name+"', try again with --force", nil)
			} else {
				logrus.WithField("image", image).Warnf("force: deleting previous image with name " + params.Name)
				err = p.DeleteImage(image.Id, true)
				if err != nil {
					return nil, errors.New("removing previously existing image", err)
				}
			}
		}
	}
	imagePath := getImagePath(params.Name)
	logrus.Debugf("making directory: %s", filepath.Dir(imagePath))
	if err := os.MkdirAll(filepath.Dir(imagePath), 0777); err != nil {
		return nil, errors.New("creating directory for boot image", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(filepath.Dir(imagePath))
		}
	}()

	logrus.WithField("raw-image", params.RawImage).Infof("copying boot volume")
	if err := unikos.CopyFile(params.RawImage.LocalImagePath, imagePath); err != nil {
		return nil, errors.New("copying qemu kernel", err)
	}

	imagePathInfo, err := os.Stat(imagePath)
	if err != nil {
		return nil, errors.New("statting raw image file", err)
	}
	sizeMb := imagePathInfo.Size() >> 20

	logrus.WithFields(logrus.Fields{
		"name": params.Name,
		"id":   params.Name,
		"size": sizeMb,
	}).Infof("creating base vmdk for unikernel image")

	image := &types.Image{
		Id:             params.Name,
		Name:           params.Name,
		RunSpec:        params.RawImage.RunSpec,
		StageSpec:      params.RawImage.StageSpec,
		SizeMb:         sizeMb,
		Infrastructure: types.Infrastructure_QEMU,
		Created:        time.Now(),
	}

	err = p.state.ModifyImages(func(images map[string]*types.Image) error {
		images[params.Name] = image
		return nil
	})
	if err != nil {
		return nil, errors.New("modifying image map in state", err)
	}
	err = p.state.Save()
	if err != nil {
		return nil, errors.New("saving image map to state", err)
	}

	logrus.WithFields(logrus.Fields{"image": image}).Infof("image created succesfully")
	return image, nil
}
