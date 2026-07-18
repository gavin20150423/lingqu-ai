<template>
  <UserWorkspaceLayout>
    <main class="community-page community-page--wide">
      <header class="community-head">
        <div>
          <p class="community-eyebrow">ACCOUNT POOL</p>
          <h1>我的账号</h1>
          <p>管理自己添加的 OpenAI / Anthropic OAuth 上游账号。</p>
        </div>
        <div class="community-actions">
          <button class="community-btn community-btn--quiet" :disabled="!selectedIds.length" @click="showBatch = true">批量编辑</button>
          <button class="community-icon-btn" title="导出账号" @click="exportAccounts"><Icon name="download" size="sm" /></button>
          <button class="community-btn community-btn--quiet" @click="showImport = true">导入</button>
          <button class="community-btn community-btn--quiet" @click="proxyPanel = true">管理代理 IP</button>
          <button class="community-btn" @click="openCreate"><Icon name="plus" size="sm" />新增账号</button>
        </div>
      </header>

      <section class="community-filterbar" aria-label="账号筛选">
        <label class="community-search">
          <Icon name="search" size="sm" />
          <input v-model="filters.search" placeholder="搜索账号名称" />
        </label>
        <select v-model="filters.provider" class="community-select"><option value="">全部平台</option><option value="openai">OpenAI</option><option value="anthropic">Anthropic</option></select>
        <select v-model="filters.type" class="community-select"><option value="">全部类型</option><option value="oauth">OAuth</option></select>
        <select v-model="filters.status" class="community-select"><option value="">全部状态</option><option value="pending">待审核</option><option value="active">正常</option><option value="paused">已暂停</option><option value="rejected">已拒绝</option></select>
        <select v-model="filters.group" class="community-select"><option value="">全部分组</option><option v-for="group in accountGroups" :key="group" :value="group">{{ group }}</option></select>
        <button class="community-icon-btn" title="刷新" :disabled="loading" @click="load"><Icon name="refresh" size="sm" /></button>
      </section>

      <div class="community-table-wrap community-account-table">
        <table class="community-table">
          <thead><tr><th><input v-model="selectAll" type="checkbox" aria-label="全选账号" /></th><th>名称</th><th>平台 / 类型</th><th>共享</th><th>容量</th><th>状态</th><th>调度</th><th>今日统计</th><th>分组</th><th>用量窗口</th><th>优先级</th><th>最近使用</th><th>过期时间</th><th>备注</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-if="loading"><td colspan="15"><div class="community-empty compact">账号加载中...</div></td></tr>
            <tr v-else-if="filteredAccounts.length === 0"><td colspan="15"><div class="community-empty"><strong>暂无个人账号</strong><span>新增账号后，可在私有或公共模式下调度使用。</span><button class="community-btn" @click="openCreate">新增账号</button></div></td></tr>
            <tr v-for="account in filteredAccounts" v-else :key="account.id">
              <td><input v-model="selectedIds" type="checkbox" :value="account.id" :aria-label="`选择 ${account.name}`" /></td>
              <td><strong>{{ account.name }}</strong><small>#{{ account.id }}</small></td>
              <td><span class="community-provider" :data-provider="account.provider">{{ account.provider === 'openai' ? 'OpenAI' : 'Anthropic' }}</span><small>{{ account.account_tier || 'OAuth' }}</small></td>
              <td><span class="community-status-dot" :class="account.share_mode">{{ account.share_mode === 'public' ? '公共' : '私有' }}</span></td>
              <td><strong>{{ account.concurrency }}</strong><small>/ {{ account.capacity }}</small></td>
              <td><span class="community-status" :class="`community-status--${account.status}`">{{ statusText(account.status) }}</span><small v-if="account.review_note" :title="account.review_note">{{ account.review_note }}</small></td>
              <td><span :class="['community-switch-label', { on: account.schedulable !== false }]">{{ account.schedulable === false ? '已暂停' : '可调度' }}</span></td>
              <td><strong>{{ account.today_requests || 0 }} 次</strong><small>{{ compactNumber(account.today_tokens || 0) }} tokens</small></td>
              <td>{{ account.group_name || '-' }}</td>
              <td><div class="community-window"><span>5h</span><progress :value="account.usage_5h_percent || 0" max="100" /><small>{{ account.usage_5h_percent || 0 }}%</small></div><div class="community-window"><span>7d</span><progress :value="account.usage_7d_percent || 0" max="100" /><small>{{ account.usage_7d_percent || 0 }}%</small></div></td>
              <td>{{ account.priority || 50 }}</td>
              <td>{{ formatTime(account.last_used_at) }}</td>
              <td>{{ formatTime(account.expires_at, '永不过期') }}</td>
              <td class="community-ellipsis" :title="account.notes">{{ account.notes || '-' }}</td>
              <td><div class="community-row-actions"><button class="community-icon-btn" title="发布共享" :disabled="account.status !== 'active'" @click="beginListing(account)"><Icon name="link" size="sm" /></button><button class="community-icon-btn danger" title="删除账号" @click="remove(account.id)"><Icon name="x" size="sm" /></button></div></td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="showCreate" class="community-modal-backdrop" @click.self="showCreate = false">
        <section class="community-modal community-modal--large" role="dialog" aria-modal="true" aria-label="添加账号">
          <header><div><p class="community-eyebrow">STEP 1 / 2</p><h2>添加账号</h2><p>先配置账号策略，保存后进入 OAuth 授权审核。</p></div><button class="community-icon-btn" title="关闭" @click="showCreate = false"><Icon name="x" size="sm" /></button></header>
          <form @submit.prevent="createAccount">
            <div class="community-form-grid">
              <label>账号名称<input v-model.trim="form.name" class="community-input" required maxlength="120" placeholder="例如：OpenAI Plus 主账号" /></label>
              <label>账号平台<select v-model="form.provider" class="community-select"><option value="openai">OpenAI OAuth</option><option value="anthropic">Anthropic OAuth</option></select></label>
              <label>共享模式<select v-model="form.share_mode" class="community-select"><option value="private">私有，仅自己调度</option><option value="public">公共，可发布到账号广场</option></select></label>
              <label v-if="form.provider === 'openai'">账号等级<select v-model="form.account_tier" class="community-select" required><option value="" disabled>请选择账号等级</option><option v-for="tier in openAITiers" :key="tier" :value="tier.toLowerCase()">{{ tier }}</option></select></label>
              <label>并发数<input v-model.number="form.concurrency" type="number" min="1" max="1000" class="community-input" /></label>
              <label>代理 IP<select v-model.number="form.proxy_id" class="community-select" :required="form.provider === 'anthropic'"><option :value="0">{{ form.provider === 'anthropic' ? '请选择代理 IP' : '无代理' }}</option><option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{ proxy.name }} · {{ proxy.protocol.toUpperCase() }}</option></select><small v-if="form.provider === 'anthropic'">Claude 授权必须选择一个可用代理。</small></label>
              <label>过期时间<input v-model="form.expires_at" type="datetime-local" class="community-input" /></label>
            </div>
            <label>备注<textarea v-model="form.notes" class="community-textarea" placeholder="可选，记录账号来源或用途" /></label>

            <fieldset class="community-fieldset">
              <legend>模型与保护</legend>
              <div v-if="form.provider === 'openai'" class="model-config">
                <div class="community-segmented"><button type="button" :class="{active:modelMode==='whitelist'}" @click="modelMode='whitelist'">模型白名单</button><button type="button" :class="{active:modelMode==='mapping'}" @click="modelMode='mapping'">模型映射</button></div>
                <template v-if="modelMode === 'whitelist'">
                  <div class="model-chip-grid"><label v-for="model in openAIModels" :key="model" class="model-chip"><input v-model="selectedModels" type="checkbox" :value="model" /><span>{{ model }}</span></label></div>
                  <div class="model-actions"><button type="button" class="community-btn community-btn--quiet" @click="selectedModels=[...openAIDefaultModels]">填入相关模型</button><button type="button" class="community-btn community-btn--quiet" @click="selectedModels=[]">清除所有模型</button></div>
                  <div class="model-custom"><input v-model.trim="customModel" class="community-input" placeholder="输入自定义模型名称" /><button type="button" class="community-btn community-btn--quiet" @click="addCustomModel">填入</button></div>
                  <small>已选择 {{ selectedModels.length }} 个模型</small>
                </template>
                <template v-else>
                  <p class="model-help">将请求模型映射到实际模型。左边是请求模型，右边是发送到上游的实际模型。</p>
                  <div class="model-mapping-presets"><button v-for="preset in modelMappingPresets" :key="preset.label" type="button" class="community-btn community-btn--quiet" @click="addModelMapping(preset.source,preset.target)">+ {{ preset.label }}</button></div>
                  <div v-for="(mapping,index) in modelMappings" :key="index" class="model-mapping-row"><input v-model.trim="mapping.source" class="community-input" placeholder="请求模型" /><span>→</span><input v-model.trim="mapping.target" class="community-input" placeholder="实际模型" /><button type="button" class="community-icon-btn danger" title="删除映射" @click="modelMappings.splice(index,1)"><Icon name="x" size="sm" /></button></div>
                  <button type="button" class="community-btn community-btn--quiet" @click="addModelMapping()">+ 添加映射</button>
                </template>
              </div>
              <label v-else>模型白名单（可选）<input v-model="modelText" class="community-input" placeholder="claude-opus-4-6, claude-sonnet-4-6" /></label>
              <div class="community-form-grid">
                <label v-if="form.provider === 'openai'">5h 限额保护（%）<input v-model.number="form.quota_5h_percent" type="number" min="1" max="100" class="community-input" /></label>
                <label v-if="form.provider === 'openai'">7d 限额保护（%）<input v-model.number="form.quota_7d_percent" type="number" min="1" max="100" class="community-input" /></label>
              </div>
              <div v-if="form.provider === 'anthropic'" class="anthropic-controls">
                <section><label class="community-check"><input v-model="form.temp_unschedulable_enabled" type="checkbox" />临时不可调度<small>错误码与关键词同时匹配时临时禁用账号</small></label><template v-if="form.temp_unschedulable_enabled"><div class="temp-rule-presets"><button type="button" class="community-btn community-btn--quiet" @click="addTempRule(529)">+ 529 过载</button><button type="button" class="community-btn community-btn--quiet" @click="addTempRule(429)">+ 429 限流</button><button type="button" class="community-btn community-btn--quiet" @click="addTempRule(503)">+ 503 维护</button><button type="button" class="community-btn community-btn--quiet" @click="addTempRule()">添加规则</button></div><div v-for="(rule,index) in form.temp_unschedulable_rules" :key="index" class="temp-rule"><label>错误码<input v-model.number="rule.error_code" type="number" min="100" max="599" class="community-input" /></label><label>关键词<input v-model="rule.keywords" class="community-input" placeholder="多个关键词用逗号分隔" /></label><label>禁用分钟<input v-model.number="rule.duration_minutes" type="number" min="1" max="1440" class="community-input" /></label><label>说明<input v-model="rule.description" class="community-input" /></label><button type="button" class="community-icon-btn danger" title="删除规则" @click="removeTempRule(index)"><Icon name="x" size="sm" /></button></div></template></section>
                <label class="community-check"><input v-model="form.intercept_warmup" type="checkbox" />拦截预热请求<small>标题生成等预热请求返回 mock 响应</small></label>
                <section><label class="community-check"><input v-model="form.window_cost_enabled" type="checkbox" />5h 窗口费用控制</label><div v-if="form.window_cost_enabled" class="community-form-grid"><label>费用阈值<input v-model.number="form.window_cost_limit" type="number" min="0.01" step="0.01" class="community-input" /><small>达到后不参与新请求调度</small></label><label>粘性预留额度<input v-model.number="form.window_cost_sticky_reserve" type="number" min="0" step="0.01" class="community-input" /></label></div></section>
                <section><label class="community-check"><input v-model="form.session_limit_enabled" type="checkbox" />会话数量控制</label><div v-if="form.session_limit_enabled" class="community-form-grid"><label>最大会话数<input v-model.number="form.session_limit" type="number" min="1" class="community-input" /></label><label>空闲超时<input v-model.number="form.session_idle_timeout_minutes" type="number" min="1" class="community-input" /><small>分钟</small></label></div></section>
                <section><label class="community-check"><input v-model="form.rpm_enabled" type="checkbox" />RPM 限制</label><div v-if="form.rpm_enabled" class="community-form-grid"><label>基础 RPM<input v-model.number="form.rpm_limit" type="number" min="1" class="community-input" /></label><label>RPM 策略<select v-model="form.rpm_strategy" class="community-select"><option value="tiered">三区模型</option><option value="sticky_exempt">粘性豁免</option></select></label><label>粘性缓冲区<input v-model.number="form.rpm_sticky_buffer" type="number" min="0" class="community-input" /><small>0 表示使用系统默认值</small></label></div></section>
                <section><strong>用户消息限速</strong><div class="community-segmented"><button type="button" :class="{active:form.user_msg_queue_mode===''}" @click="form.user_msg_queue_mode=''">关闭</button><button type="button" :class="{active:form.user_msg_queue_mode==='throttle'}" @click="form.user_msg_queue_mode='throttle'">软性限速</button><button type="button" :class="{active:form.user_msg_queue_mode==='serialize'}" @click="form.user_msg_queue_mode='serialize'">串行队列</button></div></section>
                <div class="community-toggle-grid"><label><input v-model="form.tls_fingerprint" type="checkbox" />TLS 指纹模拟</label><label><input v-model="form.session_affinity" type="checkbox" />会话 ID 伪装</label><label><input v-model="form.cache_ttl_override" type="checkbox" />缓存 TTL 强制替换</label></div>
                <label v-if="form.cache_ttl_override">目标 TTL<select v-model="form.cache_ttl_target" class="community-select"><option value="5m">5m</option><option value="1h">1h</option></select></label>
              </div>
            </fieldset>

            <fieldset class="community-fieldset">
              <legend>{{ form.provider === 'openai' ? 'OpenAI' : 'Anthropic' }} 账户授权</legend>
              <div class="oauth-flow">
                <div class="oauth-flow-step"><span>1</span><div><strong>生成官方登录链接</strong><p>在新窗口完成登录与授权。令牌只会由服务端换取并加密保存。</p></div><button type="button" class="community-btn community-btn--quiet" :disabled="authorizing || (form.provider==='anthropic'&&!form.proxy_id) || (form.provider==='openai'&&form.account_tier==='pro'&&!form.proxy_id)" @click="startOAuth">{{ authorizing ? '生成中...' : (oauthSessionId ? '重新生成' : '生成登录链接') }}</button></div>
                <div v-if="oauthAuthURL" class="oauth-auth-link"><a :href="oauthAuthURL" target="_blank" rel="noopener noreferrer">打开授权页面</a><small>完成授权后，将浏览器最终回调地址或授权结果粘贴到下一步。</small></div>
                <div class="oauth-flow-step"><span>2</span><div><strong>粘贴授权回调</strong><p>可粘贴完整 localhost 回调 URL，或仅粘贴 code；系统会自动识别。</p></div></div>
                <textarea v-model.trim="oauthCallback" class="community-textarea credential" :disabled="!oauthSessionId" placeholder="粘贴完整回调 URL，或仅粘贴 code" />
              </div>
            </fieldset>
            <footer><button type="button" class="community-btn community-btn--quiet" @click="showCreate = false">取消</button><button class="community-btn" :disabled="saving || !oauthSessionId || !oauthCallback">{{ saving ? '授权换取中...' : '完成授权并提交' }}</button></footer>
          </form>
        </section>
      </div>

      <div v-if="listing.account_id" class="community-modal-backdrop" @click.self="listing.account_id = 0">
        <section class="community-modal" role="dialog" aria-modal="true" aria-label="发布共享账号">
          <header><div><p class="community-eyebrow">OWNER TERMS</p><h2>共享条款</h2></div><button class="community-icon-btn" title="关闭" @click="listing.account_id = 0"><Icon name="x" size="sm" /></button></header>
          <form @submit.prevent="publish"><div class="community-form-grid"><label>展示名称<input v-model="listing.title" class="community-input" required /></label><label>席位数<input v-model.number="listing.seat_limit" type="number" min="1" class="community-input" /></label><label>单人并发<input v-model.number="listing.per_user_concurrency" type="number" min="1" class="community-input" /></label><label>最低余额<input v-model.number="listing.minimum_balance" type="number" min="0" step="0.01" class="community-input" /></label><label>小时费<input v-model.number="listing.hourly_price" type="number" min="0" step="0.01" class="community-input" /></label><label>免小时费低消<input v-model.number="listing.hourly_minimum_spend" type="number" min="0" step="0.01" class="community-input" /></label><label>倍率<input v-model.number="listing.usage_multiplier" type="number" min="0" step="0.0001" class="community-input" /></label><label>默认空闲退出<input v-model.number="listing.idle_timeout_minutes" type="number" min="1" max="1440" class="community-input" /></label></div><label>说明<textarea v-model="listing.description" class="community-textarea" /></label><footer><button type="button" class="community-btn community-btn--quiet" @click="listing.account_id = 0">取消</button><button class="community-btn">发布到账号广场</button></footer></form>
        </section>
      </div>

      <div v-if="showImport" class="community-modal-backdrop" @click.self="showImport = false">
        <section class="community-modal community-modal--large" role="dialog" aria-modal="true" aria-label="导入个人账号"><header><div><h2>导入个人账号</h2><p>粘贴账号凭证或导入文件，个人导入只会创建官方 OAuth 账号。</p></div><button class="community-icon-btn" title="关闭" @click="showImport = false"><Icon name="x" size="sm" /></button></header>
          <div class="import-notice">单次最多导入 100 个账号；API Key、URL、Upstream、Cookie 会被拒绝。</div>
          <div class="community-segmented import-providers"><button :class="{active:importForm.provider==='anthropic'}" @click="importForm.provider='anthropic'">Anthropic <small>Claude 官方账号</small></button><button :class="{active:importForm.provider==='openai'}" @click="importForm.provider='openai'">OpenAI <small>ChatGPT / Codex</small></button></div>
          <div v-if="importForm.provider==='openai'" class="community-form-grid"><label>OpenAI 账号等级<select v-model="importForm.account_tier" class="community-select"><option value="" disabled>请选择账号等级</option><option value="free">Free</option><option value="plus">Plus</option><option value="pro">Pro（仅账号登录）</option><option value="team">Team</option><option value="k12">K12</option></select></label><label>代理 IP<select v-model.number="importForm.proxy_id" class="community-select"><option :value="0">无代理</option><option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{ proxy.name }}</option></select></label></div>
          <div v-else class="community-form-grid"><label>代理 IP<select v-model.number="importForm.proxy_id" class="community-select" required><option :value="0">请选择代理 IP</option><option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{ proxy.name }}</option></select><small>Claude OAuth JSON 或 Session Key 必须绑定可用代理。</small></label><label>并发数<input v-model.number="importForm.concurrency" type="number" min="1" max="1000" class="community-input" /></label></div>
          <div v-if="importForm.provider==='openai'&&importForm.account_tier==='pro'" class="import-login-required"><strong>Pro 需要账号登录并选择代理 IP</strong><span>参考站不允许直接导入 Pro 凭据，请使用官方 OAuth 登录。</span><button class="community-btn" @click="openImportOAuth">账号登录导入</button></div>
          <template v-else><div class="community-segmented"><button :class="{active:importForm.mode==='text'}" @click="importForm.mode='text'">批量文本</button><button :class="{active:importForm.mode==='files'}" @click="importForm.mode='files'">文件 / 目录</button></div>
            <label v-if="importForm.mode==='text'">账号数据<textarea v-model="importForm.text" class="community-textarea credential" placeholder="普通 Token 每行一个；完整 JSON / JSON 数组可整段粘贴。" /></label>
            <div v-else class="import-file-zone"><input ref="importFileInput" type="file" accept=".json,.txt,application/json,text/plain" multiple @change="readImportFiles" /><input ref="importDirectoryInput" type="file" accept=".json,.txt,application/json,text/plain" multiple webkitdirectory directory @change="readImportFiles" /><p>已读取 {{ importFiles.length }} 个文件</p><div><button class="community-btn community-btn--quiet" @click="importFileInput?.click()">选择 JSON / TXT 文件</button><button class="community-btn community-btn--quiet" @click="importDirectoryInput?.click()">选择目录</button></div></div>
          </template>
          <footer><button class="community-btn community-btn--quiet" @click="showImport=false">取消</button><button class="community-btn" :disabled="importing||importForm.account_tier==='pro'||!hasImportSource||(importForm.provider==='openai'&&!importForm.account_tier)||(importForm.provider==='anthropic'&&!importForm.proxy_id)" @click="submitImport">{{ importing?'导入中...':'开始导入' }}</button></footer>
        </section>
      </div>

      <div v-if="showBatch" class="community-modal-backdrop" @click.self="showBatch=false"><section class="community-modal" role="dialog" aria-modal="true" aria-label="批量编辑账号"><header><div><h2>批量编辑账号</h2><p>已选择 {{ selectedIds.length }} 个账号，只修改勾选的字段。</p></div><button class="community-icon-btn" title="关闭" @click="showBatch=false"><Icon name="x" size="sm" /></button></header><form @submit.prevent="submitBatch"><div class="batch-field"><label><input v-model="batchFields.share_mode" type="checkbox" />共享模式</label><select v-model="batchForm.share_mode" class="community-select" :disabled="!batchFields.share_mode"><option value="private">私有</option><option value="public">公共</option></select></div><div class="batch-field"><label><input v-model="batchFields.concurrency" type="checkbox" />并发数</label><input v-model.number="batchForm.concurrency" type="number" min="1" max="1000" class="community-input" :disabled="!batchFields.concurrency" /></div><div class="batch-field"><label><input v-model="batchFields.priority" type="checkbox" />优先级</label><input v-model.number="batchForm.priority" type="number" min="0" max="1000" class="community-input" :disabled="!batchFields.priority" /></div><div class="batch-field"><label><input v-model="batchFields.schedulable" type="checkbox" />调度状态</label><select v-model="batchForm.schedulable" class="community-select" :disabled="!batchFields.schedulable"><option :value="true">可调度</option><option :value="false">暂停调度</option></select></div><div class="batch-field"><label><input v-model="batchFields.proxy" type="checkbox" />代理 IP</label><select v-model.number="batchForm.proxy_id" class="community-select" :disabled="!batchFields.proxy"><option :value="0">清除代理</option><option v-for="proxy in proxies" :key="proxy.id" :value="proxy.id">{{proxy.name}}</option></select></div><div class="batch-field"><label><input v-model="batchFields.notes" type="checkbox" />备注</label><input v-model="batchForm.notes" class="community-input" :disabled="!batchFields.notes" /></div><footer><button type="button" class="community-btn community-btn--quiet" @click="showBatch=false">取消</button><button class="community-btn" :disabled="batchSaving">{{batchSaving?'保存中...':'保存修改'}}</button></footer></form></section></div>

      <div v-if="proxyPanel" class="community-modal-backdrop" @click.self="proxyPanel = false">
        <section class="community-modal community-modal--large"><header><div><h2>我的代理 IP</h2><p>修改或删除自己添加的代理 IP，修改后所有关联账号会立即使用新配置。</p><small>仍被账号使用的代理不能删除，请先将这些账号替换到其它代理。</small></div><button class="community-icon-btn" title="关闭" @click="closeProxyPanel"><Icon name="x" size="sm" /></button></header>
          <div class="proxy-toolbar"><button class="community-btn" @click="openProxyEditor"><Icon name="plus" size="sm" />添加代理 IP</button></div>
          <form v-if="proxyEditorOpen" class="community-form proxy-editor" @submit.prevent="saveProxy">
            <div class="proxy-editor-head"><div><h3>{{ proxyForm.id ? '编辑代理 IP' : '添加代理 IP' }}</h3><p>使用自己的动态或静态代理</p></div><button type="button" class="community-btn community-btn--quiet" @click="resetProxyForm">取消</button></div>
            <label>智能识别（支持动态 / 静态代理 IP）<textarea v-model.trim="proxySmartInput" class="community-textarea" placeholder="192.168.0.1:8000:用户名:密码&#10;用户名:密码@192.168.0.1:8000&#10;socks5://用户名:密码@192.168.0.1:8000" /></label>
            <div><button type="button" class="community-btn community-btn--quiet" @click="parseProxyInput">识别填入</button><small class="proxy-help">支持 socks5/http/https URL，也支持账号密码前置或冒号分隔格式。</small></div>
            <div class="community-form-grid"><label>代理名称<input v-model.trim="proxyForm.name" class="community-input" placeholder="例如：Roxy 独立 IP / 家宽代理" /><small>不填会按主机和端口自动生成。</small></label><label>IP 类型<select v-model="proxyForm.ip_type" class="community-select"><option value="ipv4">IPv4</option><option value="ipv6">IPv6</option></select></label><label>协议<select v-model="proxyForm.protocol" class="community-select"><option value="socks5">SOCKS5</option><option value="socks5h">SOCKS5H</option><option value="http">HTTP</option><option value="https">HTTPS</option></select></label><label>主机<input v-model.trim="proxyForm.host" class="community-input" required placeholder="主机" /></label><label>端口<input v-model.number="proxyForm.port" class="community-input" type="number" min="1" max="65535" required placeholder="端口" /></label><label>用户名<input v-model.trim="proxyForm.username" class="community-input" placeholder="请输入用户名" /></label><label>密码<input v-model="proxyForm.password" class="community-input" type="password" :placeholder="proxyForm.id ? '留空保持原密码' : '请输入密码'" /></label></div>
            <footer><button type="button" class="community-btn community-btn--quiet" @click="resetProxyForm">取消</button><button class="community-btn">{{ proxyForm.id ? '保存修改' : '保存并使用' }}</button></footer>
          </form>
          <div class="community-list proxy-list"><div v-for="proxy in proxies" :key="proxy.id" class="proxy-row"><button type="button" class="proxy-row-main" @click="editProxy(proxy)"><strong>{{ proxy.name }}</strong><span>{{ proxy.protocol.toUpperCase() }} · {{ proxy.host }}:{{ proxy.port }} · {{ proxy.ip_type.toUpperCase() }}</span><small>{{ proxy.account_count }} 个关联账号</small></button><button type="button" class="community-icon-btn danger" title="删除代理" :disabled="proxy.account_count>0" @click="removeProxy(proxy)"><Icon name="x" size="sm" /></button></div><div v-if="proxies.length===0" class="community-empty compact"><strong>暂未添加自己的代理 IP</strong><span>使用自己的动态或静态代理</span></div></div>
        </section>
      </div>
    </main>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { communityAPI, type CommunityAccount, type CommunityProxy } from '@/api/community'
import { useAppStore } from '@/stores/app'

const app = useAppStore()
const accounts = ref<CommunityAccount[]>([])
const loading = ref(true)
const saving = ref(false)
const authorizing = ref(false)
const showCreate = ref(false)
const showImport = ref(false)
const showBatch = ref(false)
const proxyPanel = ref(false)
const importing = ref(false)
const batchSaving = ref(false)
const importFileInput = ref<HTMLInputElement>()
const importDirectoryInput = ref<HTMLInputElement>()
const importFiles = ref<Array<{ name: string; text: string }>>([])
const selectedIds = ref<number[]>([])
const modelText = ref('')
const oauthSessionId = ref('')
const oauthAuthURL = ref('')
const oauthCallback = ref('')
const proxies = ref<CommunityProxy[]>([])
const openAITiers = ['Free', 'Plus', 'Pro', 'Team', 'K12']
const openAIModels = ['codex-auto-review','gpt-4o-audio-preview','gpt-4o-realtime-preview','gpt-5.2','gpt-5.2-2025-12-11','gpt-5.2-chat-latest','gpt-5.2-pro','gpt-5.2-pro-2025-12-11','gpt-5.3-codex','gpt-5.3-codex-spark','gpt-5.4','gpt-5.4-2026-03-05','gpt-5.4-mini','gpt-5.5','gpt-5.6-luna','gpt-5.6-sol','gpt-5.6-terra','gpt-image-1','gpt-image-1.5','gpt-image-2']
const openAIDefaultModels = openAIModels.filter(model => model !== 'gpt-5.3-codex')
const modelMappingPresets = [
  { label:'GPT-4o', source:'gpt-4o', target:'gpt-4o' }, { label:'GPT-4o Mini', source:'gpt-4o-mini', target:'gpt-4o-mini' }, { label:'GPT-4.1', source:'gpt-4.1', target:'gpt-4.1' },
  { label:'o1', source:'o1', target:'o1' }, { label:'o3', source:'o3', target:'o3' }, { label:'GPT-5.3 Codex Spark', source:'gpt-5.3-codex-spark', target:'gpt-5.3-codex-spark' },
  { label:'GPT-5.2', source:'gpt-5.2', target:'gpt-5.2' }, { label:'GPT-5.6 Sol', source:'gpt-5.6-sol', target:'gpt-5.6-sol' }, { label:'GPT-5.6 Terra', source:'gpt-5.6-terra', target:'gpt-5.6-terra' },
  { label:'GPT-5.6 Luna', source:'gpt-5.6-luna', target:'gpt-5.6-luna' }, { label:'GPT-5.5', source:'gpt-5.5', target:'gpt-5.5' }, { label:'GPT-5.4', source:'gpt-5.4', target:'gpt-5.4' },
  { label:'Haiku→5.4', source:'claude-haiku-*', target:'gpt-5.4' }, { label:'Opus→5.4', source:'claude-opus-*', target:'gpt-5.4' }, { label:'Sonnet→5.4', source:'claude-sonnet-*', target:'gpt-5.4' },
]
const filters = reactive({ search: '', provider: '', type: '', status: '', group: '' })

interface TempUnschedulableRule { error_code:number; keywords:string; duration_minutes:number; description:string }
const defaultForm = () => ({ name: '', provider: 'anthropic', share_mode: 'private', account_tier: '', concurrency: 3, proxy_id: 0, expires_at: '', notes: '', quota_5h_percent: 100, quota_7d_percent: 100, temp_unschedulable_enabled: false, temp_unschedulable_rules: [] as TempUnschedulableRule[], window_cost_enabled: false, window_cost_limit: 0, window_cost_sticky_reserve: 10, session_limit_enabled: false, session_limit: 0, session_idle_timeout_minutes: 5, rpm_enabled: false, rpm_limit: 0, rpm_strategy: 'tiered', rpm_sticky_buffer: 0, user_msg_queue_mode: '', intercept_warmup: false, tls_fingerprint: false, session_affinity: false, cache_ttl_override: false, cache_ttl_target: '5m' })
const form = reactive(defaultForm())
const defaultProxyForm = () => ({ id: 0, name: '', ip_type: 'ipv4' as 'ipv4'|'ipv6', protocol: 'socks5' as 'http'|'https'|'socks5'|'socks5h', host: '', port: 0, username: '', password: '' })
const proxyForm = reactive(defaultProxyForm())
const importForm = reactive({ provider: 'openai' as 'openai'|'anthropic', account_tier: '', proxy_id: 0, concurrency: 1, mode: 'text' as 'text'|'files', text: '' })
const modelMode = ref<'whitelist'|'mapping'>('whitelist')
const selectedModels = ref<string[]>([...openAIDefaultModels])
const customModel = ref('')
const modelMappings = ref<Array<{source:string;target:string}>>([])
const proxyEditorOpen = ref(false)
const proxySmartInput = ref('')
const batchFields = reactive({ share_mode: false, concurrency: false, priority: false, schedulable: false, proxy: false, notes: false })
const batchForm = reactive({ share_mode: 'private', concurrency: 1, priority: 50, schedulable: true, proxy_id: 0, notes: '' })
const listing = reactive({ account_id: 0, title: '', description: '', seat_limit: 1, per_user_concurrency: 1, minimum_balance: 1, hourly_price: 0, hourly_minimum_spend: 0, usage_multiplier: 1, idle_timeout_minutes: 10, publish: true })

const filteredAccounts = computed(() => accounts.value.filter(account => {
  const query = filters.search.trim().toLowerCase()
  return (!query || account.name.toLowerCase().includes(query) || account.notes.toLowerCase().includes(query)) && (!filters.provider || account.provider === filters.provider) && (!filters.type || filters.type === 'oauth') && (!filters.status || account.status === filters.status) && (!filters.group || account.group_name === filters.group)
}))
const accountGroups = computed(() => [...new Set(accounts.value.map(account => account.group_name).filter((group): group is string => Boolean(group)))].sort())
const selectAll = computed({ get: () => filteredAccounts.value.length > 0 && filteredAccounts.value.every(a => selectedIds.value.includes(a.id)), set: checked => { selectedIds.value = checked ? filteredAccounts.value.map(a => a.id) : [] } })
const hasImportSource = computed(() => importForm.mode === 'text' ? Boolean(importForm.text.trim()) : importFiles.value.length > 0)

function statusText(status: string) { return ({ pending: '待审核', active: '正常', paused: '已暂停', rejected: '已拒绝', invalid: '异常' } as Record<string, string>)[status] || status }
function compactNumber(value: number) { return Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 1 }).format(value) }
function formatTime(value?: string, empty = '从未') { return value ? new Date(value).toLocaleString('zh-CN', { dateStyle: 'short', timeStyle: 'short' }) : empty }
function resetOAuth() { oauthSessionId.value = ''; oauthAuthURL.value = ''; oauthCallback.value = '' }
function openCreate() { Object.assign(form, defaultForm()); modelText.value = ''; modelMode.value = 'whitelist'; selectedModels.value = [...openAIDefaultModels]; customModel.value = ''; modelMappings.value = []; resetOAuth(); showCreate.value = true }
function openImportOAuth() { showImport.value = false; openCreate(); form.provider = 'openai'; form.account_tier = 'pro'; form.proxy_id = importForm.proxy_id }
function addCustomModel() { const model = customModel.value.trim(); if (model && !selectedModels.value.includes(model)) selectedModels.value.push(model); customModel.value = '' }
function addModelMapping(source = '', target = '') { if (source && modelMappings.value.some(item => item.source === source)) return; modelMappings.value.push({ source, target }) }
function addTempRule(code=503) { const presets:Record<number,Omit<TempUnschedulableRule,'error_code'>>={529:{keywords:'overloaded,overload',duration_minutes:10,description:'上游过载'},429:{keywords:'rate limit,too many requests',duration_minutes:10,description:'上游限流'},503:{keywords:'maintenance,unavailable',duration_minutes:10,description:'上游维护'}}; form.temp_unschedulable_rules.push({error_code:code,...(presets[code]||presets[503])}) }
function removeTempRule(index:number) { form.temp_unschedulable_rules.splice(index,1) }

async function load() { loading.value = true; try { [accounts.value, proxies.value] = await Promise.all([communityAPI.accounts(), communityAPI.proxies()]) } catch { app.showError('账号加载失败') } finally { loading.value = false } }
async function startOAuth() { if ((form.provider === 'anthropic' || form.account_tier === 'pro') && !form.proxy_id) { app.showError('请先选择代理 IP'); return } authorizing.value = true; try { const result = await communityAPI.accountOAuthURL({ provider: form.provider, proxy_id: form.proxy_id || undefined }); oauthSessionId.value = result.session_id; oauthAuthURL.value = result.auth_url; window.open(result.auth_url, '_blank', 'noopener,noreferrer'); app.showSuccess('授权页面已打开，完成后粘贴回调结果') } catch { app.showError('生成授权链接失败') } finally { authorizing.value = false } }
function parseOAuthCallback(raw: string) { const value = raw.trim(); let code = '', state = ''; try { const url = new URL(value); code = url.searchParams.get('code') || ''; state = url.searchParams.get('state') || '' } catch { const parts = value.split('#'); code = parts[0]?.trim() || ''; state = parts[1]?.trim() || '' } return { code, state } }
async function createAccount() { const callback = parseOAuthCallback(oauthCallback.value); if (!callback.code) { app.showError('回调结果缺少 code'); return } if (form.provider === 'openai' && !form.account_tier) { app.showError('请先选择 OpenAI 账号等级'); return } if (form.provider === 'anthropic' && !form.proxy_id) { app.showError('请先选择代理 IP'); return } const mapping = Object.fromEntries(modelMappings.value.filter(item=>item.source.trim()&&item.target.trim()).map(item=>[item.source.trim(),item.target.trim()])); saving.value = true; try { const { expires_at, proxy_id, ...base } = form; await communityAPI.exchangeAccountOAuth({ session_id: oauthSessionId.value, code: callback.code, state: callback.state, proxy_id: proxy_id || undefined, account: { ...base, proxy_id: proxy_id || undefined, expires_at: expires_at ? new Date(expires_at).toISOString() : undefined, capacity: form.concurrency, supported_models: form.provider === 'openai' ? selectedModels.value : modelText.value.split(',').map(v => v.trim()).filter(Boolean), provider_options: { model_mapping: mapping, quota_5h_percent: form.quota_5h_percent, quota_7d_percent: form.quota_7d_percent, temp_unschedulable_enabled: form.temp_unschedulable_enabled, temp_unschedulable_rules: form.temp_unschedulable_rules.map(rule=>({...rule,keywords:rule.keywords.split(',').map(value=>value.trim()).filter(Boolean)})), window_cost_limit: form.window_cost_enabled ? form.window_cost_limit : 0, window_cost_sticky_reserve: form.window_cost_enabled ? form.window_cost_sticky_reserve : 0, session_limit: form.session_limit_enabled ? form.session_limit : 0, session_idle_timeout_minutes: form.session_limit_enabled ? form.session_idle_timeout_minutes : 5, rpm_limit: form.rpm_enabled ? form.rpm_limit : 0, rpm_strategy: form.rpm_strategy, rpm_sticky_buffer: form.rpm_enabled ? form.rpm_sticky_buffer : 0, user_msg_queue_mode: form.user_msg_queue_mode, intercept_warmup: form.intercept_warmup, tls_fingerprint: form.tls_fingerprint, session_affinity: form.session_affinity, cache_ttl_override: form.cache_ttl_override, cache_ttl_target: form.cache_ttl_target } } }); showCreate.value = false; resetOAuth(); await load(); app.showSuccess('OAuth 账号已加密保存，等待管理员审核') } catch { app.showError('授权换取失败，请重新生成登录链接') } finally { saving.value = false } }
function resetProxyForm() { Object.assign(proxyForm, defaultProxyForm()); proxySmartInput.value = ''; proxyEditorOpen.value = false }
function openProxyEditor() { Object.assign(proxyForm, defaultProxyForm()); proxySmartInput.value = ''; proxyEditorOpen.value = true }
function closeProxyPanel() { resetProxyForm(); proxyPanel.value = false }
function editProxy(proxy: CommunityProxy) { Object.assign(proxyForm, { id: proxy.id, name: proxy.name, ip_type: proxy.ip_type, protocol: proxy.protocol, host: proxy.host, port: proxy.port, username: proxy.username, password: '' }); proxySmartInput.value = ''; proxyEditorOpen.value = true }
function normalizeColonProxy(raw:string) { const parts = raw.split(':'); if (parts.length < 2 || parts[0]?.includes('[')) return raw; const [host,port,username='',password=''] = parts; return username ? `${encodeURIComponent(username)}:${encodeURIComponent(password)}@${host}:${port}` : `${host}:${port}` }
function parseProxyInput() { const raw = proxySmartInput.value.trim(); if (!raw) return; try { const value = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `socks5://${raw.includes('@') ? raw : normalizeColonProxy(raw)}`; const parsed = new URL(value); const protocol = parsed.protocol.replace(':','').toLowerCase(); if (!['http','https','socks5','socks5h'].includes(protocol) || !parsed.hostname || !parsed.port) throw new Error('invalid'); Object.assign(proxyForm, { protocol, host: parsed.hostname, port: Number(parsed.port), username: decodeURIComponent(parsed.username), password: decodeURIComponent(parsed.password), ip_type: parsed.hostname.includes(':') ? 'ipv6' : 'ipv4' }) } catch { app.showError('无法识别代理格式，请检查主机、端口和账号密码') } }
async function saveProxy() { try { await communityAPI.saveProxy({ ...proxyForm }); resetProxyForm(); proxies.value = await communityAPI.proxies(); app.showSuccess('代理 IP 已保存') } catch { app.showError('代理保存失败，请检查协议、主机和端口') } }
async function removeProxy(proxy: CommunityProxy) { if (proxy.account_count > 0 || !confirm(`确认删除代理“${proxy.name}”？`)) return; try { await communityAPI.deleteProxy(proxy.id); proxies.value = await communityAPI.proxies(); if (proxyForm.id === proxy.id) resetProxyForm(); app.showSuccess('代理已删除') } catch { app.showError('仍被账号使用的代理不能删除') } }
async function readImportFiles(event: Event) { const input = event.target as HTMLInputElement; const files = Array.from(input.files || []).filter(file => /\.(json|txt)$/i.test(file.name)); importFiles.value = await Promise.all(files.slice(0, 100).map(async file => ({ name: file.name.replace(/\.(json|txt)$/i, ''), text: await file.text() }))); input.value = '' }
function parseImportSource(text: string, fallbackName = ''): Array<{name?:string;credential:unknown}> { const raw = text.trim(); if (!raw) return []; try { const parsed: unknown = JSON.parse(raw); const values = Array.isArray(parsed) ? parsed : [parsed]; return values.map(value => { if (value && typeof value === 'object' && !Array.isArray(value) && 'credential' in value) { const record = value as Record<string, unknown>; return { name: typeof record.name === 'string' ? record.name : (fallbackName || undefined), credential: record.credential } } return { name: values.length === 1 && fallbackName ? fallbackName : undefined, credential: value } }) } catch { return raw.split(/\r?\n/).map(value => value.trim()).filter(Boolean).map((credential, index) => ({ name: fallbackName ? `${fallbackName}${index ? ` ${index + 1}` : ''}` : undefined, credential })) } }
async function submitImport() { const sources = importForm.mode === 'text' ? parseImportSource(importForm.text) : importFiles.value.flatMap(file => parseImportSource(file.text, file.name)); if (!sources.length || sources.length > 100) { app.showError(sources.length > 100 ? '单次最多导入 100 个账号' : '没有可导入的账号数据'); return } importing.value = true; try { const result = await communityAPI.importAccounts({ provider: importForm.provider, account_tier: importForm.provider === 'openai' ? importForm.account_tier : 'oauth', proxy_id: importForm.proxy_id || undefined, concurrency: importForm.concurrency, share_mode: 'private', items: sources }); showImport.value = false; importForm.text = ''; importFiles.value = []; await load(); app.showSuccess(`成功导入 ${result.count} 个账号，等待管理员审核`) } catch { app.showError('导入失败：只接受对应平台的官方 OAuth 凭据，API Key、URL、Cookie 和 Upstream 配置会被拒绝') } finally { importing.value = false } }
async function submitBatch() { const payload: Record<string, unknown> = { ids: selectedIds.value }; if (batchFields.share_mode) payload.share_mode = batchForm.share_mode; if (batchFields.concurrency) payload.concurrency = batchForm.concurrency; if (batchFields.priority) payload.priority = batchForm.priority; if (batchFields.schedulable) payload.schedulable = batchForm.schedulable; if (batchFields.proxy) { if (batchForm.proxy_id) payload.proxy_id = batchForm.proxy_id; else payload.clear_proxy = true } if (batchFields.notes) payload.notes = batchForm.notes; if (Object.keys(payload).length === 1) { app.showError('请至少勾选一个要修改的字段'); return } batchSaving.value = true; try { const result = await communityAPI.batchUpdateAccounts(payload); showBatch.value = false; await load(); app.showSuccess(`已更新 ${result.updated} 个账号`) } catch { app.showError('批量编辑失败，请检查字段和代理归属') } finally { batchSaving.value = false } }
function beginListing(account: CommunityAccount) { Object.assign(listing, { account_id: account.id, title: account.name, seat_limit: account.capacity, per_user_concurrency: Math.min(account.concurrency, 4) }) }
async function publish() { try { await communityAPI.createListing({ ...listing }); listing.account_id = 0; await load(); app.showSuccess('共享方案已发布') } catch { app.showError('发布失败，请确认账号已审核且状态正常') } }
async function remove(id: number) { if (!confirm('确认删除这个账号？')) return; try { await communityAPI.deleteAccount(id); await load() } catch { app.showError('删除失败') } }
async function exportAccounts() { try { const exported = await communityAPI.exportAccounts(selectedIds.value); const blob = new Blob([JSON.stringify(exported, null, 2)], { type: 'application/json' }); const url = URL.createObjectURL(blob); const link = document.createElement('a'); link.href = url; link.download = `oauth-accounts-${new Date().toISOString().slice(0,10)}.json`; link.click(); URL.revokeObjectURL(url); app.showSuccess(`已导出 ${exported.length} 个账号，请妥善保管 OAuth 凭据`) } catch { app.showError('账号导出失败') } }
watch(filteredAccounts, current => { selectedIds.value = selectedIds.value.filter(id => current.some(a => a.id === id)) })
watch(() => form.provider, resetOAuth)
onMounted(load)
</script>

<style src="@/styles/community.css"></style>
<style scoped>
.oauth-flow{display:grid;gap:12px}.oauth-flow-step{display:grid;grid-template-columns:30px minmax(0,1fr) auto;align-items:center;gap:10px}.oauth-flow-step>span{display:grid;width:28px;height:28px;place-items:center;border:1px solid var(--ink);border-radius:var(--community-radius);font-weight:900}.oauth-flow-step>div{display:grid;gap:3px}.oauth-flow-step p,.oauth-auth-link small{color:var(--muted);font-size:.72rem}.oauth-auth-link{display:flex;align-items:center;justify-content:space-between;gap:12px;border:1px solid var(--line);border-radius:var(--community-radius);background:var(--community-surface-muted);padding:10px 12px}.oauth-auth-link a{font-weight:900;text-decoration:underline}.proxy-toolbar{display:flex;justify-content:flex-end}.proxy-editor{border:1px solid var(--line);border-radius:var(--community-radius-lg);background:var(--community-surface);padding:14px}.proxy-editor-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}.proxy-editor-head h3{font-size:1rem}.proxy-editor-head p,.proxy-help{color:var(--muted);font-size:.7rem}.proxy-editor>div:has(.proxy-help){display:flex;align-items:center;gap:10px;flex-wrap:wrap}.proxy-row{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:center;gap:8px;border:1px solid var(--line);border-radius:var(--community-radius);padding:8px}.proxy-row-main{display:grid;gap:3px;text-align:left}.proxy-row-main span,.proxy-row-main small{color:var(--muted);font-size:.68rem}@media(max-width:780px){.oauth-flow-step{grid-template-columns:30px minmax(0,1fr)}.oauth-flow-step>.community-btn{grid-column:1/-1}.oauth-auth-link{align-items:flex-start;flex-direction:column}}
.import-notice{border:1px solid #fedf89;border-radius:var(--community-radius);background:#fffaeb;color:#93370d;padding:10px 12px;font-size:.78rem;font-weight:700}.import-providers button{display:grid;gap:2px;text-align:left}.import-providers small{font-size:.66rem;font-weight:500;color:var(--muted)}.import-login-required{display:grid;gap:8px;border:1px solid var(--line);border-radius:var(--community-radius);padding:16px}.import-login-required span{color:var(--muted);font-size:.76rem}.import-login-required .community-btn{justify-self:start}.import-file-zone{display:grid;place-items:center;gap:12px;border:1px dashed var(--community-control-line);border-radius:var(--community-radius-lg);background:var(--community-surface-muted);padding:26px}.import-file-zone input{display:none}.import-file-zone>div{display:flex;flex-wrap:wrap;gap:8px}.batch-field{display:grid;grid-template-columns:minmax(140px,.45fr) minmax(0,1fr);align-items:center;gap:12px}.batch-field>label{display:flex;align-items:center;gap:8px;font-weight:800}@media(max-width:620px){.batch-field{grid-template-columns:1fr}.import-file-zone>div{display:grid;width:100%}}
.anthropic-controls{display:grid;gap:10px}.anthropic-controls>section,.anthropic-controls>.community-check,.anthropic-controls>label{display:grid;gap:9px;border:1px solid var(--line);padding:11px}.anthropic-controls .community-check{grid-template-columns:auto minmax(0,1fr);align-items:center;font-weight:800}.anthropic-controls .community-check small{grid-column:2;color:var(--muted);font-size:.68rem;font-weight:500}.anthropic-controls section>.community-check{border:0;padding:0}.anthropic-controls section>strong{font-size:.78rem}
.temp-rule-presets{display:flex;flex-wrap:wrap;gap:6px}.temp-rule{display:grid;grid-template-columns:90px minmax(160px,1fr) 100px minmax(130px,.7fr) auto;align-items:end;gap:7px;border-top:1px solid var(--line);padding-top:9px}.temp-rule label{display:grid;gap:4px;font-size:.7rem;font-weight:700}@media(max-width:760px){.temp-rule{grid-template-columns:1fr 1fr}.temp-rule>.community-icon-btn{align-self:end}}
.model-config{display:grid;gap:10px}.model-chip-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(190px,1fr));gap:6px}.model-chip{display:flex;align-items:center;gap:7px;border:1px solid var(--line);padding:7px 9px;font-size:.72rem}.model-actions,.model-mapping-presets{display:flex;flex-wrap:wrap;gap:6px}.model-custom,.model-mapping-row{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:center;gap:7px}.model-mapping-row{grid-template-columns:minmax(0,1fr) auto minmax(0,1fr) auto}.model-help{color:var(--muted);font-size:.72rem}@media(max-width:620px){.model-mapping-row{grid-template-columns:1fr auto 1fr auto}.model-chip-grid{grid-template-columns:1fr}}
</style>
<style scoped>
.community-modal-backdrop{z-index:10000;place-items:start center;overflow-y:auto}.community-modal{max-height:calc(100dvh - 40px)}
@media(max-width:780px){.community-modal{max-height:calc(100dvh - 16px)}}
</style>
