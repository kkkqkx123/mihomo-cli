<p align="center">
  <h1 align="center">🎭 Overwrite 覆写配置中心</h1>
  <p align="center">
    💬 <strong>欢迎使用 <a href="../../issues">Issue</a> 提出建议和问题！</strong><br>
    🔄 自动同步：<strong>每日北京时间 08:30</strong> 更新<br>
    🧩 适配建议：<strong>OpenClash v0.46.001 及以上</strong>
  </p>
  <p align="center">
    <img src="https://img.shields.io/github/last-commit/HenryChiao/mihomo_yamls?style=for-the-badge&logo=git&label=Last%20Update&color=2ea44f" />
    <img src="https://img.shields.io/github/repo-size/HenryChiao/mihomo_yamls?style=for-the-badge&logo=github&label=Repo%20Size" />
    <a href="https://github.com/vernesong/OpenClash" target="_blank">
      <img src="https://img.shields.io/badge/OpenClash-Compatible-blue?style=for-the-badge&logo=openwrt" />
    </a>
  </p>
</p>

<hr>

<h2>📖 目录</h2>
<ul>
  <li><a href="#-使用建议">📌 使用建议</a></li>
  <li><a href="#-theini--订阅转换配置">🧩 THEINI · 订阅转换配置</a>
    <ul>
      <li><a href="#什么是-ini">什么是 INI</a></li>
      <li><a href="#三大分类">三大分类</a></li>
      <li><a href="#使用方法">使用方法</a></li>
    </ul>
  </li>
  <li><a href="#️-theopenclash--覆写模块">⚙️ THEOPENCLASH · 覆写模块</a>
    <ul>
      <li><a href="#适用场景">适用场景</a></li>
      <li><a href="#配置说明">配置说明</a></li>
      <li><a href="#环境变量设置">环境变量设置</a></li>
    </ul>
  </li>
  <li><a href="#-项目来源">📂 项目来源</a></li>
</ul>

<hr>

<h2 id="-使用建议">📌 使用建议</h2>
<ul>
  <li><strong>THEINI</strong> 和 <strong>THEOPENCLASH</strong> 分别服务于不同的使用场景，请根据您的客户端类型选择：
    <ul>
      <li>使用 <strong>订阅转换网站/工具</strong> → 选择 <a href="#-theini--订阅转换配置">THEINI</a></li>
      <li>使用 <strong>OpenClash 插件</strong> → 选择 <a href="#️-theopenclash--覆写模块">THEOPENCLASH</a></li>
    </ul>
  </li>
  <li>所有配置文件均使用 <strong>GitHub Raw 直链</strong>，请确保您的设备可正常访问 GitHub 或配置相应的代理。</li>
</ul>

<hr>

<h2 id="-theini--订阅转换配置">🧩 THEINI · 订阅转换配置</h2>
<p>适用于各类基于 <a href="https://github.com/tindy2013/subconverter">SubConverter</a> 开发的订阅转换服务（如 ACL4SSR、Sub-Web 等在线转换工具）。</p>

<h3 id="什么是-ini">什么是 INI</h3>
<p>INI（Initial Configuration）是 SubConverter 的远程配置格式，定义了分流规则、策略组、节点重命名等规则。引用本目录的 INI 文件，可将机场原始订阅自动转换为完整可用的 Clash/Mihomo 配置。</p>

<h3 id="三大分类">三大分类</h3>
<table>
  <thead>
    <tr>
      <th align="left">分类目录</th>
      <th align="left">适用场景</th>
      <th align="left">特点描述</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><a href="./THEINI/ACL4Category/"><strong>ACL4Category</strong></a></td>
      <td>规则洁癖用户</td>
      <td>基于 ACL4SSR 经典规则体系，精细化分流，适合对规则质量有极高要求的用户</td>
    </tr>
    <tr>
      <td><a href="./THEINI/Airport/"><strong>Airport</strong></a></td>
      <td>特定机场用户</td>
      <td>收录 jklolixxs 等机场专属定制方案，针对特定机场节点优化分组逻辑</td>
    </tr>
    <tr>
      <td><a href="./THEINI/Ordinary/"><strong>Ordinary</strong></a></td>
      <td>通用场景</td>
      <td>ShellCrash、DustinWin 等社区主流方案，适配大多数使用场景，开箱即用</td>
    </tr>
  </tbody>
</table>

<h3 id="使用方法">使用方法</h3>
<ol>
  <li><strong>获取 Raw 链接</strong>：<br>进入上方分类目录，选择所需的 <code>.ini</code> 文件，点击 <strong>Raw</strong> 按钮复制链接。</li>
  <li><strong>填入订阅转换平台</strong>：<br>在订阅转换网站的 <strong>远程配置 (Remote Config)</strong> 栏填入上述链接，<strong>订阅链接</strong> 栏填入您的机场订阅。</li>
  <li><strong>生成配置</strong>：<br>选择目标格式（Clash / Mihomo），点击生成即可。</li>
</ol>
<blockquote>
  <p>💡 <strong>提示</strong>：建议使用 Raw 直链而非下载文件，以确保每日自动同步的最新规则生效。</p>
</blockquote>

<hr>

<h2 id="️-theopenclash--覆写模块">⚙️ THEOPENCLASH · 覆写模块</h2>
<p>专为 <strong>OpenClash 插件</strong> 设计的覆写配置文件（<code>.conf</code>），支持自动注入多订阅源与定时更新。</p>

<h3 id="适用场景">适用场景</h3>
<ul>
  <li><strong>主路由/软路由用户</strong>：直接通过 OpenClash 插件一键导入，无需手动下载 YAML</li>
  <li><strong>多机场/多订阅用户</strong>：支持 <code>EN_KEY1</code>, <code>EN_KEY2</code>, <code>EN_KEY3</code> 批量注入多个 <code>proxy-providers</code></li>
  <li><strong>自动化维护</strong>：内置 Cron 定时任务，每日自动更新配置</li>
</ul>

<h3 id="配置说明">配置说明</h3>
<table>
  <thead>
    <tr>
      <th align="left">分类目录</th>
      <th align="left">对应内核</th>
      <th align="left">说明</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><a href="./THEOPENCLASH/General_Config/"><strong>General_Config</strong></a></td>
      <td>标准 Mihomo/Meta</td>
      <td>通用进阶配置的覆写脚本，适合日常使用</td>
    </tr>
    <tr>
      <td><a href="./THEOPENCLASH/Smart_Mode/"><strong>Smart_Mode</strong></a></td>
      <td><strong>Smart 内核</strong></td>
      <td>SmartDNS/软路由专用，需启用 Smart 内核</td>
    </tr>
    <tr>
      <td><a href="./THEOPENCLASH/Mobile_Modules/"><strong>Mobile_Modules</strong></a></td>
      <td>标准内核</td>
      <td>Android 模块场景的覆写脚本</td>
    </tr>
  </tbody>
</table>

<h3 id="环境变量设置">使用方法</h3>

<h4>1️⃣ 新增覆写模块</h4>
<p>进入 OpenClash → <strong>配置订阅</strong> → <strong>添加</strong>：</p>
<ul>
  <li><strong>配置名称</strong>：自定义（如 "Henry-Smart"）</li>
  <li><strong>订阅方式</strong>：<code>http</code></li>
  <li><strong>订阅地址</strong>：选择对应分类下的 <code>.conf</code> 文件 Raw 链接</li>
</ul>

<p><strong>示例链接</strong>：</p>
<pre><code>https://raw.githubusercontent.com/HenryChiao/mihomo_yamls/main/Overwrite/THEOPENCLASH/Smart_Mode/HenryChiao/MihomoSmartAIO.conf</code></pre>

<h4>2️⃣ 配置环境变量</h4>
<p>在 OpenClash <strong>覆写设置</strong> 或系统环境变量中添加：</p>
<pre><code>EN_KEY1=https://你的机场订阅链接1;EN_KEY2=https://你的机场订阅链接2;EN_KEY3=https://你的机场订阅链接3</code></pre>

<p><strong>变量说明</strong>：</p>
<table>
  <thead>
    <tr>
      <th align="left">变量名</th>
      <th align="left">必填</th>
      <th align="left">说明</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><code>EN_KEY1</code></td>
      <td>✅ 是</td>
      <td>第一个订阅源（主机场）</td>
    </tr>
    <tr>
      <td><code>EN_KEY2</code></td>
      <td>❌ 否</td>
      <td>第二个订阅源（备用/分流）</td>
    </tr>
    <tr>
      <td><code>EN_KEY3</code></td>
      <td>❌ 否</td>
      <td>第三个订阅源（按配置需求）</td>
    </tr>
  </tbody>
</table>

<p><strong>配置完成后</strong>：保存 → 点击 <strong>应用配置</strong> 或重启 OpenClash。</p>

<h4>3️⃣ 工作原理</h4>
<pre>
OpenClash 启动
    ↓
下载 .conf 覆写脚本
    ↓
读取 EN_KEY1/2/3 环境变量
    ↓
自动替换 proxy-providers 中的 URL 占位符
    ↓
注入真实订阅链接
    ↓
启动 Mihomo 内核（自动生成完整配置）
</pre>

<hr>

<h2>💡 温馨提示</h2>
<ol>
  <li><strong>网络连通性</strong>：本仓库文件均使用 GitHub Raw 直链，请确保路由器可正常访问 GitHub，或在 OpenClash 中配置 <strong>GitHub 加速</strong>。</li>
  <li><strong>GeoIP 数据库</strong>：初次使用若提示内核错误，通常是由于缺少 GeoIP 数据库，请手动更新 <strong>GeoIP Dat</strong> 或多次重启后自动恢复。</li>
  <li><strong>Smart 内核</strong>：使用 <code>THEOPENCLASH/Smart_Mode/</code> 时，请确保 OpenClash 已切换至 <strong>Smart 内核</strong>，否则策略可能无法正常加载。</li>
</ol>

<hr>

<h2 id="-项目来源">📂 项目来源</h2>
<ul>
  <li><strong>THEYAMLS</strong>：原始配置文件同步来源（HenryChiao、666OS、JohnsonRan 等上游作者）</li>
  <li><strong>THEINI upstream</strong>：ACL4SSR、ShellCrash、DustinWin、jklolixxs 等社区规则项目</li>
  <li><strong>OpenClash</strong>：Vernesong 开发的 OpenWrt 代理插件</li>
</ul>

<hr>

<p align="center">
  ✨ <strong>如果本项目对您有帮助，请点个 ⭐ Star 支持！</strong><br>
  配置问题欢迎 <a href="../../issues">Issue</a> 交流，优质建议欢迎 PR！<br><br>
  <strong>🔙 <a href="../README.md">返回项目主页</a></strong>
</p>
