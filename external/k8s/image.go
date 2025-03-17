package k8s

import "strings"

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

// get image from registry/repository:tag
func (i *Image) getFromImageName(name string) {
	s := strings.Split(name, ":")
	if len(s) == 1 {
		i.Repository = name
		i.Tag = "latest"
		return
	}

	i.Repository = s[0]
	i.Tag = s[1]
}

func NewImage(s string) *Image {
	image := &Image{}
	image.getFromImageName(s)
	return image
}
