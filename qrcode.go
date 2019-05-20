package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/LyricTian/logger"
	"github.com/nfnt/resize"
	"go-qrcode-1"
)

var (
	text      string
	logo      string
	percent   int
	size      int
	start     int
	end       int
	worker    int
	out       string
	TaskQueue []chan *QrcodeConf
	total     int
	totalNow  int
)

type QrcodeConf struct {
	text string
	out  string
}

func init() {
	flag.StringVar(&text, "t", "请认准四特酒有限责任公司郑州酒乐邦食品有限公司专供酒水\n客服电话：13526560403\n编号：%06d", "二维码内容")
	flag.StringVar(&logo, "l", "data/logo.jpg", "二维码Logo(png)")
	flag.IntVar(&percent, "p", 40, "二维码Logo的显示比例(默认15%)")
	flag.IntVar(&start, "start", 200, "开始编号")
	flag.IntVar(&end, "end", 220, "结束编号")
	flag.IntVar(&worker, "worker", 10, "工作线程")
	flag.IntVar(&size, "s", 2400, "二维码的大小(默认256)")
	flag.StringVar(&out, "o", "output", "输出文件")
}

func StartWorkerPool() {
	TaskQueue = make([]chan *QrcodeConf, worker)
	for i := 0; i < int(worker); i++ {
		TaskQueue[i] = make(chan *QrcodeConf, 10)
		go startOneWorker(i, TaskQueue[i])
	}
}

func startOneWorker(workerId int, confs chan *QrcodeConf) {
	fmt.Println("Start worker ", workerId)
	for {
		select {
		case request := <-confs:
			genQrcode(request.text, request.out)
		}
	}
}

func main() {
	flag.Parse()

	if text == "" {
		logger.Errorf("请指定二维码的生成内容")
	}

	if out == "" {
		logger.Errorf("请指定输出文件")
	}

	if exists, err := checkFile(out); err != nil {
		logger.Errorf("检查输出文件发生错误：%s", err.Error())
	} else if exists {
		//logger.Errorf("输出文件已经存在，请重新指定")
	}

	tmpI := 0
	tmpDirStart := 0
	tmpDirEnd := 0

	StartWorkerPool()

	total = end - start

	for i := start; i < end; i++ {
		tmpI = i + 1
		tmpDirStart = i/500*500 + 1
		tmpDirEnd = i/500 + 500
		tmpConf := &QrcodeConf{
			text: fmt.Sprintf(text, tmpI),
			out:  out + "/ouput_" + strconv.Itoa(tmpDirStart) + "_" + strconv.Itoa(tmpDirEnd) + "/qrcode_" + strconv.Itoa(tmpI) + ".jpg",
		}
		TaskQueue[i%worker] <- tmpConf
	}

	for {
		time.Sleep(10 * time.Second)
		if total == totalNow {
			logger.Infof("gen end %d-%d", start, end)
			os.Exit(0)
		} else {
			logger.Infof("gen process %.2d", totalNow/total)
		}
	}
}

func genQrcode(text string, out string) {
	code, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		logger.Errorf("创建二维码发生错误：%s", err.Error())
		return
	}

	srcImage := code.Image(size)
	if logo != "" {
		logoSize := float64(size) * float64(percent) / 100

		srcImage, err = addLogo(srcImage, logo, int(logoSize))
		if err != nil {
			logger.Errorf("增加Logo发生错误：%s", err.Error())
			return
		}
	}

	outAbs, err := filepath.Abs(out)
	if err != nil {
		logger.Errorf("获取输出文件绝对路径发生错误：%s", out)
		return
	}

	os.MkdirAll(filepath.Dir(outAbs), 0777)
	outFile, err := os.Create(outAbs)
	if err != nil {
		logger.Errorf("创建输出文件发生错误：%s", err.Error())
		return
	}
	defer outFile.Close()

	jpeg.Encode(outFile, srcImage, &jpeg.Options{Quality: 100})

	logger.Infof("qrcode gen success：%s", outAbs)

	totalNow++
}

func checkFile(name string) (bool, error) {
	_, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func resizeLogo(logo string, size uint) (image.Image, error) {
	file, err := os.Open(logo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	img = resize.Resize(size, size, img, resize.Lanczos3)
	return img, nil
}

func addLogo(srcImage image.Image, logo string, size int) (image.Image, error) {
	logoImage, err := resizeLogo(logo, uint(size))
	if err != nil {
		return nil, err
	}

	offset := image.Pt((srcImage.Bounds().Dx()-logoImage.Bounds().Dx())/2, (srcImage.Bounds().Dy()-logoImage.Bounds().Dy())/2)
	b := srcImage.Bounds()
	m := image.NewNRGBA(b)
	draw.Draw(m, b, srcImage, image.ZP, draw.Src)
	draw.Draw(m, logoImage.Bounds().Add(offset), logoImage, image.ZP, draw.Over)

	return m, nil
}
