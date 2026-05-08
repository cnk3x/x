package urlx

func DirPath(rawUrl string) string {
	return Parse(rawUrl).DirPath()
}

func FileName(rawUrl string) string {
	return Parse(rawUrl).FileName()
}

func DirName(rawUrl string) string {
	return Parse(rawUrl).DirName()
}

func FileExt(rawUrl string) string {
	return Parse(rawUrl).FileExt()
}
