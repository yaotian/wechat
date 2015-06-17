package entry

import (
	"encoding/xml"
	"errors"
	"time"
)

type CDATAText struct {
	Text string `xml:",innerxml"`
}

func value2CDATA(v string) CDATAText {
	return CDATAText{"<![CDATA[" + v + "]]>"}
}

type Response struct {
	ToUserName   CDATAText
	FromUserName CDATAText
	MsgType      CDATAText
	CreateTime   time.Duration
}

type TextResponse struct {
	XMLName xml.Name `xml:"xml"`
	Response
	Content CDATAText
}

type ImageResponse struct {
	XMLName xml.Name `xml:"xml"`
	Response
	MediaId string `xml:"Image>MediaId"`
}

type VoiceResponse struct {
	XMLName xml.Name `xml:"xml"`
	Response
	MediaId string `xml:"Voice>MediaId"`
}

type VideoResponse struct {
	XMLName xml.Name `xml:"xml"`
	Response
	MediaId     string `xml:"Video>MediaId"`
	Title       string `xml:"Video>Title"`
	Description string `xml:"Video>Description"`
}

type MusicResponse struct {
	XMLName xml.Name `xml:"xml"`
	Response
	Title        string `xml:"Music>Title"`
	Description  string `xml:"Music>Description"`
	MusicUrl     string `xml:"Music>MusicUrl"`
	HQMusicUrl   string `xml:"Music>HQMusicUrl"`
	ThumbMediaId string `xml:"Music>ThumbMediaId"`
}

type NewsResponse struct {
	XMLName xml.Name `xml:"xml"`
	Response
	ArticleCount int
	News         Articles `xml:"Articles"`
}

func NewTextResponse(from string, to string, content string) *TextResponse {
	text := new(TextResponse)
	text.FromUserName = value2CDATA(from)
	text.ToUserName = value2CDATA(to)
	text.MsgType = value2CDATA("text")
	text.Content = value2CDATA(content)
	text.CreateTime = time.Duration(time.Now().Unix())
	return text
}

func NewImageResponse(from string, to string, media string) *ImageResponse {
	image := new(ImageResponse)
	image.FromUserName = value2CDATA(from)
	image.ToUserName = value2CDATA(to)
	image.MediaId = media
	image.MsgType = value2CDATA("image")
	image.CreateTime = time.Duration(time.Now().Unix())
	return image
}

func NewVoiceResponse(from string, to string, media string) *VoiceResponse {
	voice := new(VoiceResponse)
	voice.FromUserName = value2CDATA(from)
	voice.ToUserName = value2CDATA(to)
	voice.MediaId = media
	voice.MsgType = value2CDATA("voice")
	voice.CreateTime = time.Duration(time.Now().Unix())
	return voice
}

func NewVideoResponse(from string, to string, media string, title string, description string) *VideoResponse {
	video := new(VideoResponse)
	video.FromUserName = value2CDATA(from)
	video.ToUserName = value2CDATA(to)
	video.MediaId = media
	video.Title = title
	video.Description = description
	video.MsgType = value2CDATA("video")
	video.CreateTime = time.Duration(time.Now().Unix())
	return video
}

func NewMusicResponse(from, to, title, description, musicurl, hqmusicurl, thumb string) *MusicResponse {
	music := new(MusicResponse)
	music.FromUserName = value2CDATA(from)
	music.ToUserName = value2CDATA(to)
	music.MsgType = value2CDATA("music")
	music.Title = title
	music.Description = description
	music.MusicUrl = musicurl
	music.HQMusicUrl = hqmusicurl
	music.ThumbMediaId = thumb
	return music
}

func NewNewsResponse(from string, to string) *NewsResponse {
	news := new(NewsResponse)
	news.FromUserName = value2CDATA(from)
	news.ToUserName = value2CDATA(to)
	news.MsgType = value2CDATA("news")
	news.ArticleCount = 0
	return news
}

func (news *NewsResponse) Append(article *Article) error {
	if len(news.News.Item) >= 10 {
		return errors.New("entry NewsResponse: news response append exceed 10 articles already.")
	}

	news.News.Item = append(news.News.Item, article)
	news.ArticleCount = len(news.News.Item)
	return nil
}
