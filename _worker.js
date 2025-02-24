var __defProp = Object.defineProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });

// src/index.js
var src_default = {
  async fetch(request) {
    const url = new URL(request.url);
    const domain = (url.searchParams.get("domain") || "example.com").trim();
    let html = `
		<!DOCTYPE html>
		<html>
		<head>
		  <title>DNS \u67E5\u8BE2\u5DE5\u5177</title>
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
			  <input type="text" name="domain" value="${domain}" placeholder="\u8F93\u5165\u57DF\u540D">
			  <button type="submit">\u67E5\u8BE2</button>
			</form>
		  </div>
		  <h2>${domain} \u7684 DNS \u8BB0\u5F55</h2>
		  <table>
			<tr>
			  <th>\u5E8F\u53F7</th>
			  <th>\u7C7B\u578B</th>
			  <th>\u503C</th>
			  <th>\u7269\u7406\u4F4D\u7F6E</th>
			</tr>
	  `;
    try {
      const dnsQuery = /* @__PURE__ */ __name(async (type) => {
        const response = await fetch(`https://dns.google.com/resolve?name=${domain}&type=${type}`, {
          headers: {
            accept: "application/dns-json"
          }
        });
        return await response.json();
      }, "dnsQuery");
      const aRecords = await dnsQuery("A");
      const aaaaRecords = await dnsQuery("AAAA");
      let rowNumber = 1;
      if (aRecords.Answer) {
        for (const record of aRecords.Answer) {
          const ip = record.data;
          let location = await getLocation(ip);
          html += `
			  <tr>
				<td>${rowNumber}</td>
				<td>A</td>
				<td>${ip}</td>
				<td>${location || "\u65E0\u6CD5\u83B7\u53D6\u4F4D\u7F6E\u4FE1\u606F"}</td>
			  </tr>
			`;
          rowNumber++;
        }
      }
      if (aaaaRecords.Answer) {
        for (const record of aaaaRecords.Answer) {
          const ip = record.data;
          let location = await getLocation(ip);
          html += `
			  <tr>
				<td>${rowNumber}</td>
				<td>AAAA</td>
				<td>${ip}</td>
				<td>${location || "\u65E0\u6CD5\u83B7\u53D6\u4F4D\u7F6E\u4FE1\u606F"}</td>
			  </tr>
			`;
          rowNumber++;
        }
      }
      if (!aRecords.Answer && !aaaaRecords.Answer) {
        html += `
			<tr>
			  <td colspan="4">\u672A\u627E\u5230 A \u6216 AAAA \u8BB0\u5F55</td>
			</tr>
		  `;
      }
    } catch (error) {
      html += `
		  <tr>
			<td colspan="4">\u67E5\u8BE2\u51FA\u9519: ${error.message}</td>
		  </tr>
		`;
    }
    html += `
		  </table>
		  <p>\u6CE8\u610F\uFF1A\u5728Workers Free\u4E0A\uFF0C\u6BCF\u4E2A\u8BF7\u6C42\u53EF\u4EE5\u53D1\u51FA50\u4E2A\u5B50\u8BF7\u6C42\uFF0CA\u8BB0\u5F55\u67E5\u8BE2\u548CAAAA\u8BB0\u5F55\u67E5\u8BE2+48\u6761\u7269\u7406\u4F4D\u7F6E\u67E5\u8BE2=50\u4E2A\u5B50\u8BF7\u6C42\u3002</p>
		</body>
		</html>
	  `;
    return new Response(html, {
      headers: { "content-type": "text/html;charset=UTF-8" }
    });
  }
};
async function getLocation(ip) {
  try {
    const response = await fetch(`http://ip-api.com/json/${ip}`);
    const data = await response.json();
    if (data.status === "success") {
      return `${data.city || ""}, ${data.regionName || ""}, ${data.country || ""}`;
    }
    return null;
  } catch (error) {
    return null;
  }
}
__name(getLocation, "getLocation");
export {
  src_default as default
};
//# sourceMappingURL=index.js.map
