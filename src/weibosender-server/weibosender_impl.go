package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	utils "github.com/ewangplay/go-utils"
	"io"
	"io/ioutil"
	"jzlservice/weibosender"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type WeiboErrorInfo struct {
	Error_code int64  `json:"error_code"`
	Error      string `json:"error"`
}

func ParseErrMsg(result []byte) error {
	var info WeiboErrorInfo
	err := json.Unmarshal(result, &info)
	if err == nil {
		if info.Error_code != 0 {
			return fmt.Errorf("%v: %v", info.Error_code, info.Error)
		}
	}
	return nil
}

// WeiboSenderImpl implementaion
type WeiboSenderImpl struct {
}

func (this *WeiboSenderImpl) Ping() (r string, err error) {
	LOG_INFO("请求ping方法")
	return "pong", nil
}

func (this *WeiboSenderImpl) SendStatus(status *weibosender.WeiboStatus) (r string, err error) {
	LOG_INFO("微博[%v]开始发送", status.Status)

	weibo_status := utils.UrlEncode(status.Status)

	var requestUrl string
	var resp *http.Response

	if status.Pic != "" {

		var b bytes.Buffer
		formdata := multipart.NewWriter(&b)
		formdata.WriteField("access_token", status.AccessToken)
		formdata.WriteField("status", weibo_status)
		formdata.WriteField("visible", strconv.FormatInt(int64(status.Visible), 10))
		formdata.WriteField("list_id", status.ListId)
		formdata.WriteField("lat", strconv.FormatFloat(status.Latitude, 'f', -1, 64))
		formdata.WriteField("long", strconv.FormatFloat(status.Longitude, 'f', -1, 64))
		formdata.WriteField("annotations", status.Annotations)
		formdata.WriteField("rip", status.RealIp)

		picdata, _ := formdata.CreateFormFile("pic", status.Pic)
		if strings.HasPrefix(status.Pic, "http") {
			res, err := http.Get(status.Pic)
			if err != nil {
				LOG_ERROR("获取图片[%v]失败. 失败原因：%v", status.Pic, err)
				return "", err
			}
			io.Copy(picdata, res.Body)
			res.Body.Close()
		} else {
			fh, _ := os.Open(status.Pic)
			io.Copy(picdata, fh)
			fh.Close()
		}
		form_type := formdata.FormDataContentType()

		formdata.Close()

		resp, err = http.Post("https://upload.api.weibo.com/2/statuses/upload.json", form_type, &b)
		if err != nil {
			LOG_ERROR("请求发送微博失败. 失败原因：%v", err)
			return "", err
		}
		defer resp.Body.Close()

	} else {

		if status.Visible == 3 && status.ListId != "" {
			requestUrl = fmt.Sprintf("https://api.weibo.com/2/statuses/update.json?access_token=%s&status=%s&visible=%v&list_id=%v&lat=%v&long=%v&annotations=%v&rip=%v",
				status.AccessToken,
				weibo_status,
				status.Visible,
				status.ListId,
				status.Latitude,
				status.Longitude,
				status.Annotations,
				status.RealIp)

		} else {
			requestUrl = fmt.Sprintf("https://api.weibo.com/2/statuses/update.json?access_token=%s&status=%s&lat=%v&long=%v&annotations=%v&rip=%v",
				status.AccessToken,
				weibo_status,
				status.Latitude,
				status.Longitude,
				status.Annotations,
				status.RealIp)
		}

		LOG_DEBUG("最终的请求URL：%v", requestUrl)

		req, err := http.NewRequest("POST", requestUrl, nil)
		if err != nil {
			LOG_ERROR("创建HTTP POST请求失败. 失败原因：%v", err)
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
		httpClient := &http.Client{}
		resp, err = httpClient.Do(req)
		if err != nil {
			LOG_ERROR("请求微博发送URL地址[%v]失败. 失败原因：%v", requestUrl, err)
			return "", err
		}
		defer resp.Body.Close()
	}

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取HTTP响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("HTTP请求响应信息：%v", result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("微博[%v]发送失败: %v", status.Status, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("微博[%v]发送失败。Http状态码：%v", status.Status, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("微博[%v]发送成功", status.Status)

	return result, nil
}

func (this *WeiboSenderImpl) SendMessage(access_token string, type_a1 string, data string, receiver_id int64, save_sender_box int32) (r string, err error) {
	LOG_INFO("回复用户[%v]的私信消息开始", receiver_id)

	var requestUrl string
	requestUrl = fmt.Sprintf("https://m.api.weibo.com/2/messages/reply.json?access_token=%v&type=%v&data=%v&receiver_id=%v&save_sender_box=%v",
		access_token,
		type_a1,
		data,
		receiver_id,
		save_sender_box)

	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("回复用户[%v]的私信消息的请求URL: %v", receiver_id, requestUrl)

	req, err := http.NewRequest("POST", requestUrl, nil)
	if err != nil {
		LOG_ERROR("创建HTTP POST请求失败. 失败原因：%v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		LOG_ERROR("请求私信发送URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取HTTP响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("回复用户[%v]的私信消息的请求状态：%v", receiver_id, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("回复用户[%v]的私信消息失败: %v", receiver_id, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("回复用户[%v]的私信消息失败。Http状态码：%v", receiver_id, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("回复用户[%v]的私信消息成功", receiver_id)

	return result, nil
}

// Parameters:
//  - AccessToken
//  - Uid
func (this *WeiboSenderImpl) GetUserInfoById(access_token string, uid int64) (r string, err error) {
	LOG_INFO("获取用户[%v]的信息开始", uid)

	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/users/show.json?access_token=%v&uid=%v", access_token, uid)

	//parse the url and encode the values
	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("获取用户[%v]的信息的请求URL: %v", uid, requestUrl)

	resp, err := http.Get(requestUrl)
	if err != nil {
		LOG_ERROR("请求获取用户信息的URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取HTTP响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("获取用户[%v]的信息：%v", uid, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("获取用户[%v]的信息失败: %v", uid, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("获取用户[%v]的信息失败。Http状态码：%v", uid, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("获取用户[%v]的信息成功", uid)

	return result, nil
}

//Parameters:
//  - AccessToken
//  - EmotionType
func (this *WeiboSenderImpl) GetEmotions(access_token, emotion_type string) (r string, err error) {

	LOG_INFO("更新微博表情开始")
	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/emotions.json?access_token=%v&type=%v&language=cnname", access_token, emotion_type)
	//parse the url and encode the values
	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("获取类型为[%v]的表情请求URL: %v", emotion_type, requestUrl)

	resp, err := http.Get(requestUrl)
	if err != nil {
		LOG_ERROR("请求获取表情的URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取HTTP响应信息失败. 失败原因：%v", err)
		return "", err
	}
	result = string(body)

	LOG_DEBUG("获取表情类型[%v]的信息：%v", emotion_type, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("获取表情类型[%v]的信息失败: %v", emotion_type, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("获取表情类型[%v]的信息失败。Http状态码：%v", emotion_type, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("获取表情类型[%v]的信息成功", emotion_type)

	return result, nil
}

// Parameters:
//  - AccessToken
//  - ScreenName
func (this *WeiboSenderImpl) GetUserInfoByName(access_token string, screen_name string) (r string, err error) {
	LOG_INFO("获取用户[%v]的信息开始", screen_name)

	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/users/show.json?access_token=%v&screen_name=%v", access_token, screen_name)

	//parse the url and encode the values
	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("获取用户[%v]的信息的请求URL: %v", screen_name, requestUrl)

	resp, err := http.Get(requestUrl)
	if err != nil {
		LOG_ERROR("请求获取用户信息的URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取HTTP响应信息失败. 失败原因：%v", err)
		return "", err
	}
	result = string(body)

	LOG_DEBUG("获取用户[%v]的信息：%v", screen_name, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("获取用户[%v]的信息失败: %v", screen_name, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("获取用户[%v]的信息失败。Http状态码：%v", screen_name, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("获取用户[%v]的信息成功", screen_name)

	return result, nil
}

// @描述:
//   获取用户最新发布的微博
//
// @参数:
//   access_token: 通过OAuth2认证得到的身份标识
//   since_id: 若指定此参数，则返回ID比since_id大的微博（即比since_id时间晚的微博），默认为0
//   max_id: 若指定此参数，则返回ID小于或等于max_id的微博，默认为0
//   count: 单页返回的记录条数，最大不超过100，超过100以100处理，默认为20
//   page: 返回结果的页码，默认为1
//
// @返回:
//   请求成功返回最新发布的微博列表；请求失败返回空
//
func (this *WeiboSenderImpl) GetStatuses(access_token string, since_id int64, max_id int64, count int32, page int32) (r string, err error) {
	LOG_INFO("获取最近发布的微博列表开始")

	//组装请求的URL
	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/statuses/user_timeline.json?access_token=%v&since_id=%v&max_id=%v&count=%v&page=%v",
		access_token,
		since_id,
		max_id,
		count,
		page)

	//对请求的URL做Url-Encode编码
	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("获取最近发布的微博列表的请求URL：%v", requestUrl)

	//向目标地址发起请求
	resp, err := http.Get(requestUrl)
	if err != nil {
		LOG_ERROR("请求获取最近发布的微博列表URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取Http响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("获取最近发布的微博列表：%v", result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("获取最近发布的微博列表失败: %v", err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("获取最近发布的微博列表失败。Http状态码：%v", resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("获取最近发布的微博列表成功")

	return result, nil
}

// @描述:
//   获取用户及其关注用户最新发布的微博
//
// @参数:
//   access_token: 通过OAuth2认证得到的身份标识
//   since_id: 若指定此参数，则返回ID比since_id大的微博（即比since_id时间晚的微博），默认为0
//   max_id: 若指定此参数，则返回ID小于或等于max_id的微博，默认为0
//   count: 单页返回的记录条数，最大不超过100，超过100以100处理，默认为20
//   page: 返回结果的页码，默认为1
//
// @返回:
//   请求成功返回用户及其关注用户最新发布的微博列表；请求失败返回空
//
func (this *WeiboSenderImpl) GetConcernStatuses(access_token string, since_id int64, max_id int64, count int32, page int32) (r string, err error) {
	LOG_INFO("获取用户[%v]关注用户的最近发布的微博列表开始", access_token)

	//组装请求的URL
	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/statuses/home_timeline.json?access_token=%v&since_id=%v&max_id=%v&count=%v&page=%v",
		access_token,
		since_id,
		max_id,
		count,
		page)

	//对请求的URL做Url-Encode编码
	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("获取用户[%v]关注用户的最近发布的微博列表的请求URL：%v", access_token, requestUrl)

	//向目标地址发起请求
	resp, err := http.Get(requestUrl)
	if err != nil {
		LOG_ERROR("请求获取最近发布的微博列表URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取Http响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("获取用户[%v]关注用户的最近发布的微博列表的响应信息：%v", access_token, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("获取最近发布的微博列表失败: %v", err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("获取最近发布的微博列表失败。Http状态码：%v", resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("获取用户[%v]关注用户的最近发布的微博列表成功", access_token)

	return result, nil
}

// @描述:
//  获取微博的评论数、转发数和点赞数（点赞数暂不支持获取）
//
// @参数:
//   access_token: 通过OAuth2认证得到的身份标识
//   ids: 需要获取数据的微博ID，多个之间用逗号分隔，最多不超过100个
//
// @返回:
//   请求成功返回对应微博ID的评论数、转发数和点赞数列表；请求失败返回空
//
func (this *WeiboSenderImpl) GetStatusInteractCount(access_token string, ids string) (r string, err error) {
	LOG_INFO("获取微博[%v]的互动数开始", ids)

	//组装请求的URL
	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/statuses/count.json?access_token=%v&ids=%v",
		access_token,
		ids)

	//对请求的URL做Url-Encode编码
	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("获取微博[%v]的互动数的请求URL：%v", ids, requestUrl)

	//向目标地址发起请求
	resp, err := http.Get(requestUrl)
	if err != nil {
		LOG_ERROR("请求[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取Http响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("获取微博[%v]的互动数响应信息：%v", ids, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("获取微博[%v]的互动数失败: %v", ids, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("获取微博[%v]的互动数失败。Http状态码：%v", ids, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("获取微博[%v]的互动数成功", ids)

	return result, nil
}

func (this *WeiboSenderImpl) CreateComments(access_token string, comments string, msg_id int64, is_comment_ori int32) (r string, err error) {
	LOG_INFO("评论微博[%v]开始", msg_id)

	encodedComments := utils.UrlEncode(comments)

	var requestUrl string
	requestUrl = fmt.Sprintf("https://api.weibo.com/2/comments/create.json?access_token=%v&comment=%v&id=%v&comment_ori=%v",
		access_token,
		encodedComments,
		msg_id,
		is_comment_ori)

	urlObj, err := url.Parse(requestUrl)
	if err != nil {
		LOG_ERROR("解析URL地址[%v]失败. 失败原因：%v", requestUrl, err)
		return "", err
	}
	requestUrl = fmt.Sprintf("%v://%v%v?%v", urlObj.Scheme, urlObj.Host, urlObj.Path, urlObj.Query().Encode())

	LOG_DEBUG("评论微博[%v]的请求URL: %v", msg_id, requestUrl)

	req, err := http.NewRequest("POST", requestUrl, nil)
	if err != nil {
		LOG_ERROR("创建HTTP POST请求失败. 失败原因：%v", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		LOG_ERROR("请求评论微博[%v]的URL地址[%v]失败. 失败原因：%v", msg_id, requestUrl, err)
		return "", err
	}
	defer resp.Body.Close()

	var result string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOG_ERROR("读取HTTP响应信息失败. 失败原因：%v", err)
		return "", err
	}

	result = string(body)

	LOG_DEBUG("评论微博[%v]的请求状态：%v", msg_id, result)

	err = ParseErrMsg(body)
	if err != nil {
		LOG_ERROR("评论微博[%v]失败: %v", msg_id, err)
		return "", err
	}

	if resp.StatusCode != 200 {
		LOG_ERROR("评论微博[%v]失败。Http状态码：%v", msg_id, resp.StatusCode)
		return "", fmt.Errorf("Http Status Code: %v", resp.StatusCode)
	}

	LOG_INFO("评论微博[%v]成功", msg_id)

	return result, nil
}

func encodeMultipart(pic string) (multipartContentType string, multipartData *bytes.Buffer, err error) {
	multipartData = new(bytes.Buffer)
	formdata := multipart.NewWriter(multipartData)
	defer formdata.Close()

	picdata, _ := formdata.CreateFormFile("pic", pic)
	if strings.HasPrefix(pic, "http") {
		res, err := http.Get(pic)
		if err != nil {
			return "", nil, err
		}
		io.Copy(picdata, res.Body)
		res.Body.Close()
	} else {
		fh, _ := os.Open(pic)
		io.Copy(picdata, fh)
		fh.Close()
	}
	multipartContentType = formdata.FormDataContentType()

	return multipartContentType, multipartData, nil
}
