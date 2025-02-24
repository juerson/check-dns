export default {
	async fetch(request) {
		// 获取查询参数中的域名并去除左右空白
		const url = new URL(request.url);
		const domain = (url.searchParams.get('domain') || 'example.com').trim();

		// HTML 模板
		let html = `
		<!DOCTYPE html>
		<html>
		<head>
		  <title>DNS 查询工具</title>
		  <style>
			body { 
			  font-family: Arial, sans-serif; 
			  margin: 0 auto; 
			  max-width: 1000px; 
			  padding: 20px;
			  display: flex;
			  flex-direction: column;
			  align-items: center;
			}
			table { 
			  border-collapse: collapse; 
			  width: 100%; 
			  max-width: 800px; 
			  margin-top: 20px;
			}
			th, td { 
			  border: 1px solid #ddd; 
			  padding: 8px; 
			  text-align: left; 
			}
			th { 
			  background-color: #f2f2f2; 
			}
			.input-container { 
			  margin-bottom: 20px; 
			  width: 100%;
			  max-width: 800px;
			  text-align: center;
			}
			input[type="text"] {
			  padding: 5px;
			  width: 60%;
			}
			button {
			  padding: 5px 15px;
			  margin-left: 10px;
			}
		  </style>
		</head>
		<body>
		  <div class="input-container">
			<form method="GET">
			  <input type="text" name="domain" value="${domain}" placeholder="输入域名">
			  <button type="submit">查询</button>
			</form>
		  </div>
		  <h2>${domain} 的 DNS 记录</h2>
		  <table>
			<tr>
			  <th>序号</th>
			  <th>类型</th>
			  <th>值</th>
			  <th>物理位置</th>
			</tr>
	  `;

		try {
			// 使用 Google 的 DNS over HTTPS API 查询 A 和 AAAA 记录
			const dnsQuery = async (type) => {
				const response = await fetch(`https://dns.google.com/resolve?name=${domain}&type=${type}`, {
					headers: {
						accept: 'application/dns-json',
					},
				});
				return await response.json();
			};

			// 查询 A 记录 (IPv4) 和 AAAA 记录 (IPv6)
			const aRecords = await dnsQuery('A');
			const aaaaRecords = await dnsQuery('AAAA');

			let rowNumber = 1; // 序号计数器

			// 处理 A 记录
			if (aRecords.Answer) {
				for (const record of aRecords.Answer) {
					const ip = record.data;
					let location = await getLocation(ip);
					html += `
			  <tr>
				<td>${rowNumber}</td>
				<td>A</td>
				<td>${ip}</td>
				<td>${location || '无法获取位置信息'}</td>
			  </tr>
			`;
					rowNumber++;
				}
			}

			// 处理 AAAA 记录
			if (aaaaRecords.Answer) {
				for (const record of aaaaRecords.Answer) {
					const ip = record.data;
					let location = await getLocation(ip);
					html += `
			  <tr>
				<td>${rowNumber}</td>
				<td>AAAA</td>
				<td>${ip}</td>
				<td>${location || '无法获取位置信息'}</td>
			  </tr>
			`;
					rowNumber++;
				}
			}

			// 如果没有任何记录
			if (!aRecords.Answer && !aaaaRecords.Answer) {
				html += `
			<tr>
			  <td colspan="4">未找到 A 或 AAAA 记录</td>
			</tr>
		  `;
			}
		} catch (error) {
			html += `
		  <tr>
			<td colspan="4">查询出错: ${error.message}</td>
		  </tr>
		`;
		}

		// 结束 HTML
		html += `
		  </table>
		  <p>注意：在Workers Free上，每个请求可以发出50个子请求，A记录查询和AAAA记录查询+48条物理位置查询=50个子请求。</p>
		</body>
		</html>
	  `;

		return new Response(html, {
			headers: { 'content-type': 'text/html;charset=UTF-8' },
		});
	},
};

// 获取 IP 的物理位置
async function getLocation(ip) {
	try {
		const response = await fetch(`http://ip-api.com/json/${ip}`);
		const data = await response.json();

		if (data.status === 'success') {
			return `${data.city || ''}, ${data.regionName || ''}, ${data.country || ''}`;
		}
		return null;
	} catch (error) {
		return null;
	}
}