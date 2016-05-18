package rump

import (
	"fmt"
	"strings"

	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/emc-advanced-dev/unik/pkg/types"
)

func CreateImageVirtualBox(kernel string, args string, mntPoints []string) (*types.RawImage, error) {

	// create rump config
	var c rumpConfig

	if args == "" {
		c = setRumpCmdLine(c, "program.bin", nil)
	} else {
		c = setRumpCmdLine(c, "program.bin", strings.Split(args, " "))
	}

	res := &types.RawImage{}
	// add root -> sd0 mapping
	res.RunSpec.DeviceMappings = append(res.RunSpec.DeviceMappings,
		types.DeviceMapping{MountPoint: "/", DeviceName: "sd0"})

	for i, mntPoint := range mntPoints {
		deviceMapped := fmt.Sprintf("sd%ca", '1'+i)
		blk := blk{
			Source:     "dev",
			Path:       "/dev/" + deviceMapped,
			FSType:     "blk",
			MountPoint: mntPoint,
		}

		c.Blk = append(c.Blk, blk)
		logrus.Debugf("adding mount point to image: %s:%s", mntPoint, deviceMapped)
		res.RunSpec.DeviceMappings = append(res.RunSpec.DeviceMappings,
			types.DeviceMapping{MountPoint: mntPoint, DeviceName: deviceMapped})
	}

	// virtualbox network
	c.Net = &net{
		If:     "vioif0",
		Type:   "inet",
		Method: DHCP,
	}
	c.Net1 = &net{
		If:     "vioif1",
		Type:   "inet",
		Method: DHCP,
	}

	cmdline, err := toRumpJson(c)
	if err != nil {
		return nil, err
	}

	r, err := regexp.Compile("net[1-9]")
	if err != nil {
		return nil, err
	}
	cmdline = string(r.ReplaceAll([]byte(cmdline), []byte("net")))

	logrus.Debugf("writing rump json config: %s", cmdline)

	imgFile, err := BuildBootableImage(kernel, cmdline)
	if err != nil {
		return nil, err
	}

	res.LocalImagePath = imgFile
	res.StageSpec.ImageFormat = types.ImageFormat_RAW
	res.RunSpec.StorageDriver = types.StorageDriver_SCSI
	res.RunSpec.DefaultInstanceMemory = 512
	return res, nil

}
