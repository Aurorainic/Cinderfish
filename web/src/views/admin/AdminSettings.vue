<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { api } from "../../api/client";
import AppIcon from "../../components/AppIcon.vue";
import { instance, loadInstance } from "../../stores/instance";
import AdminCallout from "./AdminCallout.vue";
import {
  FEDERATION_MODES,
  federationModeOf,
  registrationLabel,
  type FederationMode,
} from "./meta";

const mode = ref<FederationMode | "">("");
const modeSaving = ref(false);
const modeNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);

const hosts = ref<string[]>([]);
const hostsLoading = ref(true);
const hostsError = ref("");
const newHost = ref("");
const hostPending = ref(false);
const hostNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);
const removing = ref<string | null>(null);
const reloading = ref(false);

const allowlistActive = computed(() => mode.value === "allowlist");
const registration = computed(() => registrationLabel(instance.meta));
const registrationOpen = computed(
  () => instance.meta !== null && !instance.meta.disableRegistration,
);

function labelOf(id: FederationMode): string {
  return FEDERATION_MODES.find((item) => item.id === id)?.label ?? id;
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

watch(
  () => instance.meta,
  () => {
    if (mode.value) return;
    const federation = instance.meta?.federation;
    if (federation) mode.value = federationModeOf(federation);
  },
  { immediate: true },
);

async function reloadMeta(): Promise<void> {
  modeNotice.value = null;
  await loadInstance();
  const federation = instance.meta?.federation;
  if (federation) mode.value = federationModeOf(federation);
  else modeNotice.value = { kind: "bad", text: "无法读取实例信息，请检查网络后重试。" };
}

async function selectMode(next: FederationMode) {
  if (!mode.value || modeSaving.value || next === mode.value) return;
  modeSaving.value = true;
  modeNotice.value = null;
  try {
    const res = await api.federationMode(next);
    const saved = federationModeOf(res.mode || next);
    mode.value = saved;
    modeNotice.value = { kind: "ok", text: `已切换为「${labelOf(saved)}」。` };
  } catch (err) {
    modeNotice.value = { kind: "bad", text: `切换失败：${messageOf(err)}` };
  } finally {
    modeSaving.value = false;
  }
}

async function loadHosts() {
  if (reloading.value) return;
  reloading.value = true;
  hostsLoading.value = true;
  hostsError.value = "";
  try {
    const res = await api.federationAllowlist();
    hosts.value = res.hosts ?? [];
  } catch (err) {
    hostsError.value = `白名单加载失败：${messageOf(err)}`;
    hosts.value = [];
  } finally {
    hostsLoading.value = false;
    reloading.value = false;
  }
}

const HOST_PATTERN = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$/;

async function addHost() {
  if (!allowlistActive.value || hostPending.value) return;
  hostNotice.value = null;
  const host = newHost.value.trim().toLowerCase();
  if (!host) {
    hostNotice.value = { kind: "bad", text: "请输入要添加的实例域名。" };
    return;
  }
  if (!HOST_PATTERN.test(host)) {
    hostNotice.value = { kind: "bad", text: `「${host}」不是有效的实例域名。` };
    return;
  }
  if (hosts.value.includes(host)) {
    hostNotice.value = { kind: "bad", text: `「${host}」已经在白名单中。` };
    return;
  }
  hostPending.value = true;
  try {
    const res = await api.updateFederationAllowlist(host, false);
    hosts.value = res.hosts ?? [];
    newHost.value = "";
    hostNotice.value = { kind: "ok", text: `已添加 ${host}。` };
  } catch (err) {
    hostNotice.value = { kind: "bad", text: `添加失败：${messageOf(err)}` };
  } finally {
    hostPending.value = false;
  }
}

async function removeHost(host: string) {
  if (!allowlistActive.value || removing.value !== null) return;
  hostNotice.value = null;
  removing.value = host;
  try {
    const res = await api.updateFederationAllowlist(host, true);
    hosts.value = res.hosts ?? [];
    hostNotice.value = { kind: "ok", text: `已移除 ${host}。` };
  } catch (err) {
    hostNotice.value = { kind: "bad", text: `移除失败：${messageOf(err)}` };
  } finally {
    removing.value = null;
  }
}

onMounted(() => {
  void loadHosts();
  void loadInstance().then(() => {
    if (!mode.value && !instance.meta) {
      modeNotice.value = { kind: "bad", text: "无法读取实例信息，请检查网络后重试。" };
    }
  });
});
</script>

<template>
  <section class="card">
    <div class="section-head">
      <h2>联邦模式</h2>
      <span v-if="modeSaving" class="hint">正在保存…</span>
    </div>
    <p class="muted small section-desc">
      决定本实例与其他实例互通的程度。切换后立即生效，无需重启。
    </p>

    <div class="modes" role="group" aria-label="联邦模式">
      <button
        v-for="item in FEDERATION_MODES"
        :key="item.id"
        type="button"
        class="mode"
        :class="{ active: mode === item.id }"
        :aria-pressed="mode === item.id"
        :disabled="!mode"
        @click="selectMode(item.id)"
      >
        <span class="mode-name">
          {{ item.label }}
          <span v-if="mode === item.id" class="mode-tag">
            <AppIcon name="check" />
            当前
          </span>
        </span>
        <span class="mode-desc">{{ item.desc }}</span>
      </button>
    </div>

    <p
      v-if="modeNotice"
      class="notice"
      :class="modeNotice.kind === 'ok' ? 'info' : 'error'"
      role="status"
      aria-live="polite"
    >
      {{ modeNotice.text }}
    </p>
    <p v-else-if="!mode" class="notice info">
      <span>正在读取当前联邦模式…</span>
      <button class="button ghost small" type="button" @click="reloadMeta">重新加载</button>
    </p>

    <AdminCallout v-if="mode === 'none'" title="联邦已关闭" tone="warn">
      <p>来自其他实例的笔记与活动将不再接收，本实例的笔记也不会投递出去。</p>
    </AdminCallout>
  </section>

  <section class="card">
    <div class="section-head">
      <h2>联邦白名单</h2>
      <span v-if="hosts.length" class="count">{{ hosts.length }}</span>
    </div>
    <p class="muted small section-desc">
      按域名后缀匹配：加入 <code>example.com</code> 后，<code>example.com</code> 与子域
      <code>sub.example.com</code> 都会被允许，而 <code>notexample.com</code> 不会被允许。
    </p>

    <AdminCallout
      v-if="mode && !allowlistActive"
      class="settings-alert"
      title="白名单当前不生效"
      tone="danger"
    >
      <p>当前联邦模式不是「仅白名单」，白名单中的域名不会被使用。切换到「仅白名单」后才会生效。</p>
    </AdminCallout>

    <form class="row-form" :class="{ dimmed: !allowlistActive }" @submit.prevent="addHost">
      <label class="sr-only" for="allowlist-host">要添加的实例域名</label>
      <input
        id="allowlist-host"
        v-model="newHost"
        class="input"
        type="text"
        placeholder="example.com"
        autocomplete="off"
        spellcheck="false"
        :disabled="!allowlistActive || hostPending"
      />
      <button class="button" type="submit" :disabled="!allowlistActive || hostPending">
        {{ hostPending ? "添加中…" : "添加" }}
      </button>
    </form>

    <p
      v-if="hostNotice"
      class="notice"
      :class="hostNotice.kind === 'ok' ? 'info' : 'error'"
      role="status"
      aria-live="polite"
    >
      {{ hostNotice.text }}
    </p>

    <div v-if="hostsError" class="notice error" role="alert">
      <span>{{ hostsError }}</span>
      <button class="button ghost small" type="button" @click="loadHosts">重试</button>
    </div>

    <p v-if="hostsLoading" class="hint">正在加载白名单…</p>
    <ul v-else-if="hosts.length" class="hosts" :class="{ dimmed: !allowlistActive }">
      <li v-for="host in hosts" :key="host" class="host">
        <span class="host-name">{{ host }}</span>
        <button
          type="button"
          class="button ghost small remove"
          :disabled="!allowlistActive || removing !== null"
          :aria-label="`从白名单移除 ${host}`"
          @click="removeHost(host)"
        >
          {{ removing === host ? "移除中…" : "移除" }}
        </button>
      </li>
    </ul>
    <p v-else-if="!hostsError" class="hint">
      {{
        allowlistActive
          ? "白名单为空：仅白名单模式下，本实例不会与任何其他实例互通。"
          : "白名单为空。"
      }}
    </p>

    <p class="hint format-hint">只填域名，不需要写协议或路径。</p>
  </section>

  <section class="card">
    <div class="section-head">
      <h2>注册</h2>
      <span class="pill" :class="{ open: registrationOpen }">
        {{ instance.meta ? (registrationOpen ? "自助注册开启" : "自助注册关闭") : "读取中…" }}
      </span>
    </div>
    <p class="muted small section-desc">
      当前识别到的模式：<strong>{{ registration }}</strong>。注册模式由服务端
      <code>registration_mode</code> 配置决定，修改后需要重启服务生效；当前版本没有提供在线切换接口。
    </p>

    <AdminCallout
      v-if="instance.meta && !registrationOpen"
      title="自助注册已关闭"
      tone="warn"
    >
      <p>新账号只能由管理员在「用户」页创建；如果实例启用了邀请模式，也可以凭邀请码注册。</p>
      <div class="callout-actions">
        <RouterLink :to="{ path: '/admin', query: { tab: 'users' } }" class="button ghost small">
          去创建账号
        </RouterLink>
      </div>
    </AdminCallout>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  margin-bottom: var(--space-6);
}

.section-head h2 {
  font-size: var(--fs-md);
}

.section-desc {
  margin-bottom: var(--space-14);
}

.count {
  display: inline-grid;
  place-items: center;
  min-width: 22px;
  height: 22px;
  padding: 0 var(--space-6);
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: var(--fs-xs);
  font-weight: var(--weight-bold);
}

.pill {
  padding: 2px var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text-weak);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-bold);
}

.pill.open {
  border-color: var(--success);
  background: var(--success-soft);
  color: var(--success);
}

.modes {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  margin-bottom: var(--space-14);
}

.mode {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: var(--space-12);
  padding: var(--space-12) var(--space-14);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-inset);
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--dur-base) ease, background var(--dur-base) ease,
    opacity var(--dur-base) ease;
}

.mode:hover:not(:disabled) {
  border-color: var(--accent);
}

.mode:focus-visible {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--focus-halo);
}

.mode.active {
  border-color: var(--accent);
  background: var(--accent-soft);
}

.mode:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.mode-name {
  display: flex;
  align-items: center;
  flex: none;
  gap: var(--space-8);
  min-width: 92px;
  font-weight: var(--weight-bold);
  color: var(--text-strong);
}

.mode-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px var(--space-8) 1px var(--space-6);
  border-radius: var(--radius-full);
  background: var(--accent);
  color: var(--text-on-accent);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-bold);
}

.mode-tag .icon {
  width: 0.9em;
  height: 0.9em;
}

.mode-desc {
  flex: 1;
  min-width: 0;
  font-size: var(--fs-sm);
  color: var(--text-weak);
  line-height: var(--leading-tight);
}

.settings-alert {
  margin-bottom: var(--space-14);
}

.row-form {
  display: flex;
  gap: var(--space-8);
  margin-bottom: var(--space-14);
}

.row-form .input {
  flex: 1;
  min-width: 0;
}

.dimmed {
  opacity: 0.55;
}

.hosts {
  list-style: none;
  margin: 0 0 var(--space-4);
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.host {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-12);
  padding: var(--space-8) var(--space-8) var(--space-8) var(--space-14);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-inset);
}

.host-name {
  font-weight: var(--weight-semibold);
  overflow-wrap: anywhere;
}

.remove {
  color: var(--danger);
}

.remove:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}

.format-hint {
  margin-top: var(--space-10);
}

code {
  padding: 1px var(--space-4);
  border-radius: var(--radius-xs);
  background: var(--panel-2);
  font-size: 0.88em;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 620px) {
  .mode {
    flex-direction: column;
    gap: var(--space-6);
  }

  .row-form {
    flex-direction: column;
  }
}
</style>
