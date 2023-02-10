package inter

import "context"

func DownloadThumbnail(ctx context.Context, netconn INet, fio IFs, filename string, imgUrl string) (string, error) {
	data, err := netconn.GetFile(ctx, imgUrl)
	if err != nil {
		return "", err
	}
	coverFile := filename + ".jpg"
	err = fio.Save(coverFile, data)
	if err != nil {
		return "", err
	}
	return coverFile, nil
}
