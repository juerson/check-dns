本代码库用于部署在cloudflare workers/pages的代码。

#### 用途：

查询指定域名的dns记录情况（只能查A记录和AAAA记录）。

注意：查询依赖 Google 的 DNS over HTTPS API，查询结果对应的值可能出错（可能出现cname记录），自己理性判断。还有查询物理位置时，只能查询到前面48个，后面的物理位置查询不了，会显示“无法获取位置信息”，如若要查询，需要付费版的 workers（共有1000次子请求）。

#### 截图：

<img src="images\1.png" />