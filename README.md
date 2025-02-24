本代码库用于部署在cloudflare workers/pages的代码。

#### 用途：

查询指定域名的dns记录情况（只能查A记录和AAAA记录）。

注意：查询依赖 Google 的 DNS over HTTPS API，查询结果对应的值可能出错（可能出现cname记录），自己理性判断。

#### 截图：

<img src="images\1.png" />