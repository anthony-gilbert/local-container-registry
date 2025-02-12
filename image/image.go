package image

type Image struct {
	id          int64
	CommitSHA   string
	ImageID     string
	Author      string
	TimeStamp   string
	ImageSize   string
	Description string
	ImageTag    string
}

type Images []Image

func (image *Images) Add(id int64) {

}
