<template>
  <UserWorkspaceLayout>
    <main class="community-page">
      <header class="community-head"><div><p class="community-eyebrow">SUPPORT INBOX</p><h1>工单服务</h1><p>与管理员沟通问题、接收通知并查看处理进度。</p></div><button class="community-btn" @click="openCreate"><Icon name="plus" size="sm" />发起工单</button></header>
      <section class="community-filterbar ticket-filters" aria-label="工单筛选"><label class="community-search"><Icon name="search" size="sm" /><input v-model="filters.search" placeholder="搜索主题或消息" /></label><select v-model="filters.status" class="community-select"><option value="">全部状态</option><option value="open">处理中</option><option value="waiting_admin">等待管理员</option><option value="waiting_user">等待回复</option><option value="resolved">已解决</option><option value="closed">已关闭</option></select><select v-model="filters.message" class="community-select"><option value="">全部消息</option><option value="unread">有未读消息</option><option value="mine">等待我的回复</option></select><button class="community-icon-btn" title="刷新" @click="load"><Icon name="refresh" size="sm" /></button></section>
      <div class="ticket-count"><span>工单服务</span><strong>{{ filteredTickets.length }}</strong></div>
      <section class="community-split ticket-layout">
        <aside class="community-list ticket-list"><div v-if="filteredTickets.length === 0" class="community-empty"><strong>暂无工单</strong><span>发起工单后，消息会固定在这一条工单中。</span><button class="community-btn" @click="openCreate">发起工单</button></div><button v-for="item in filteredTickets" v-else :key="item.id" :class="{ active: selected?.id === item.id }" @click="open(item.id)"><div class="community-row"><strong>{{ item.subject }}</strong><span v-if="item.user_unread" class="ticket-unread">{{ item.user_unread }}</span></div><span>{{ categoryText(item.category) }} · {{ priorityText(item.priority) }}</span><small>#{{ item.id }} · {{ statusText(item.status) }} · {{ format(item.updated_at) }}</small></button></aside>
        <article v-if="selected" class="ticket-thread"><header><div><p class="community-eyebrow">TICKET #{{ selected.id }}</p><h2>{{ selected.subject }}</h2><p>{{ categoryText(selected.category) }} · {{ priorityText(selected.priority) }} · 创建于 {{format(selected.created_at)}}</p></div><div class="ticket-head-actions"><span class="community-status" :class="`community-status--${selected.status}`">{{ statusText(selected.status) }}</span><button class="community-btn community-btn--quiet" @click="markRead">标记已读</button><button v-if="selected.status!=='closed'" class="community-btn community-btn--danger" @click="closeSelected">关闭工单</button></div></header><div class="community-messages"><div v-for="message in selected.messages" :key="message.id" class="community-message" :class="message.author_role"><small>{{ message.author_role === 'admin' ? '客服' : message.author_role === 'system' ? '系统' : '我' }} · {{ format(message.created_at) }}</small><p>{{ message.content }}</p></div></div><form v-if="selected.status !== 'closed'" class="ticket-reply" @submit.prevent="reply"><textarea v-model.trim="replyText" class="community-textarea" placeholder="输入回复内容..." required /><button class="community-btn">发送</button></form></article>
        <div v-else class="community-empty"><strong>选择一条工单</strong><span>查看完整对话与处理进度。</span></div>
      </section>
      <div v-if="creating" class="community-modal-backdrop" @click.self="creating = false"><section class="community-modal" role="dialog" aria-modal="true" aria-label="发起工单"><header><div><p class="community-eyebrow">NEW TICKET</p><h2>发起工单</h2></div><button class="community-icon-btn" title="关闭" @click="creating = false"><Icon name="x" size="sm" /></button></header><form @submit.prevent="create"><label>主题<input v-model.trim="form.subject" class="community-input" required /></label><div class="community-form-grid"><label>类型<select v-model="form.category" class="community-select"><option value="support">支持</option><option value="account">账号问题</option><option value="billing">账单与提现</option><option value="store">商城订单</option><option value="suggestion">建议</option></select></label><label>优先级<select v-model="form.priority" class="community-select"><option value="low">低</option><option value="normal">普通</option><option value="high">较急</option><option value="urgent">紧急</option></select></label></div><label>内容<textarea v-model.trim="form.content" class="community-textarea" required /></label><footer><button type="button" class="community-btn community-btn--quiet" @click="creating = false">取消</button><button class="community-btn">发起工单</button></footer></form></section></div>
    </main>
  </UserWorkspaceLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import UserWorkspaceLayout from '@/components/layout/UserWorkspaceLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { communityAPI, type SupportTicket } from '@/api/community'
import { useAppStore } from '@/stores/app'
const app=useAppStore(),tickets=ref<SupportTicket[]>([]),selected=ref<SupportTicket|null>(null),creating=ref(false),replyText=ref('')
const filters=reactive({search:'',status:'',message:''})
const form=reactive({subject:'工单服务',category:'support',priority:'normal',content:''})
const filteredTickets=computed(()=>tickets.value.filter(ticket=>{const q=filters.search.trim().toLowerCase();const messages=(ticket.messages||[]).map(m=>m.content).join(' ').toLowerCase();return(!q||ticket.subject.toLowerCase().includes(q)||messages.includes(q))&&(!filters.status||ticket.status===filters.status)&&(!filters.message||(filters.message==='unread'?ticket.user_unread>0:ticket.status==='waiting_user'))}))
const format=(value:string)=>new Date(value).toLocaleString('zh-CN',{dateStyle:'short',timeStyle:'short'})
const statusText=(value:string)=>({open:'处理中',waiting_admin:'等待管理员',waiting_user:'等待回复',resolved:'已解决',closed:'已关闭'} as Record<string,string>)[value]||value
const categoryText=(value:string)=>({support:'支持',general:'支持',account:'账号问题',billing:'账单与提现',store:'商城订单',suggestion:'建议'} as Record<string,string>)[value]||value
const priorityText=(value:string)=>({low:'低',normal:'普通',high:'较急',urgent:'紧急'} as Record<string,string>)[value]||value
function openCreate(){Object.assign(form,{subject:'工单服务',category:'support',priority:'normal',content:''});creating.value=true}
async function load(){try{tickets.value=await communityAPI.tickets()}catch{app.showError('工单加载失败')}}
async function open(id:number){try{selected.value=await communityAPI.ticket(id)}catch{app.showError('工单详情加载失败')}}
async function create(){try{const ticket=await communityAPI.createTicket(form);creating.value=false;await load();await open(ticket.id)}catch{app.showError('提交工单失败')}}
async function reply(){if(!selected.value)return;try{await communityAPI.replyTicket(selected.value.id,replyText.value);replyText.value='';await open(selected.value.id);await load()}catch{app.showError('消息发送失败')}}
async function markRead(){if(!selected.value)return;try{await communityAPI.markTicketRead(selected.value.id);selected.value.user_unread=0;await load();app.showSuccess('已标记为已读')}catch{app.showError('标记已读失败')}}
async function closeSelected(){if(!selected.value||!confirm('确认关闭这条工单？'))return;try{await communityAPI.closeTicket(selected.value.id);await open(selected.value.id);await load();app.showSuccess('工单已关闭')}catch{app.showError('关闭工单失败')}}
onMounted(load)
</script>
<style src="@/styles/community.css"></style>
<style scoped>.ticket-head-actions{display:flex;align-items:center;justify-content:flex-end;flex-wrap:wrap;gap:7px}.ticket-head-actions .community-status{margin-right:4px}@media(max-width:720px){.ticket-thread>header{display:grid}.ticket-head-actions{justify-content:flex-start}}</style>
