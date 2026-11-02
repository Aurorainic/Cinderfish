<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { api } from "../../api/client";
import AppIcon from "../../components/AppIcon.vue";
import MkButton from "../../components/ui/MkButton.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import MkInput from "../../components/ui/MkInput.vue";
import MkLoading from "../../components/ui/MkLoading.vue";
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
  <section class="_card">
    <div class="_title section-head">
      <h2>联邦模式</h2>
      <MkLoading v-if="modeSaving" em :size-em="0.8" />
    </div>
    <div class="_content">
      <p class="_caption section-desc">
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

      <MkInfo
        v-if="modeNotice"
        class="notice-row"
        :warn="modeNotice.kind === 'bad'"
        role="status"
        aria-live="polite"
      >
        {{ modeNotice.text }}
      </MkInfo>
      <MkInfo v-else-if="!mode" class="notice-row">
        <span>正在读取当前联邦模式…</span>
        <MkButton @click="reloadMeta">重新加载</MkButton>
      </MkInfo>

      <AdminCallout v-if="mode === 'none'" class="callout-row" title="联邦已关闭" tone="warn">
        <p>来自其他实例的笔记与活动将不再接收，本实例的笔记也不会投递出去。</p>
      </AdminCallout>
    </div>
  </section>

  <section class="_card">
    <div class="_title section-head">
      <h2>联邦白名单</h2>
      <span v-if="hosts.length" class="count">{{ hosts.length }}</span>
    </div>
    <div class="_content">
      <p class="_caption section-desc">
        按域名后缀匹配：加入 <code>example.com</code> 后，<code>example.com</code> 与子域
        <code>sub.example.com</code> 都会被允许，而 <code>notexample.com</code> 不会被允许。
      </p>

      <AdminCallout
        v-if="mode && !allowlistActive"
        class="callout-row"
        title="白名单当前不生效"
        tone="danger"
      >
        <p>当前联邦模式不是「仅白名单」，白名单中的域名不会被使用。切换到「仅白名单」后才会生效。</p>
      </AdminCallout>

      <form class="row-form" :class="{ dimmed: !allowlistActive }" @submit.prevent="addHost">
        <MkInput
          v-model="newHost"
          class="row-input"
          type="text"
          placeholder="example.com"
          autocomplete="off"
          :spellcheck="false"
          :disabled="!allowlistActive || hostPending"
        >
          <template #label>实例域名</template>
        </MkInput>
        <MkButton type="submit" :disabled="!allowlistActive || hostPending">
          {{ hostPending ? "添加中…" : "添加" }}
        </MkButton>
      </form>

      <MkInfo
        v-if="hostNotice"
        class="notice-row"
        :warn="hostNotice.kind === 'bad'"
        role="status"
        aria-live="polite"
      >
        {{ hostNotice.text }}
      </MkInfo>

      <MkInfo v-if="hostsError" class="notice-row" warn role="alert">
        <span>{{ hostsError }}</span>
        <MkButton @click="loadHosts">重试</MkButton>
      </MkInfo>

      <MkLoading v-if="hostsLoading" />
      <ul v-else-if="hosts.length" class="hosts" :class="{ dimmed: !allowlistActive }">
        <li v-for="host in hosts" :key="host" class="host">
          <span class="host-name">{{ host }}</span>
          <MkButton
            mini
            danger
            :disabled="!allowlistActive || removing !== null"
            :aria-label="`从白名单移除 ${host}`"
            @click="removeHost(host)"
          >
            {{ removing === host ? "移除中…" : "移除" }}
          </MkButton>
        </li>
      </ul>
      <p v-else-if="!hostsError" class="_caption">
        {{
          allowlistActive
            ? "白名单为空：仅白名单模式下，本实例不会与任何其他实例互通。"
            : "白名单为空。"
        }}
      </p>

      <p class="_caption format-hint">只填域名，不需要写协议或路径。</p>
    </div>
  </section>

  <section class="_card">
    <div class="_title section-head">
      <h2>注册</h2>
      <span class="pill" :class="{ open: registrationOpen }">
        {{ instance.meta ? (registrationOpen ? "自助注册开启" : "自助注册关闭") : "读取中…" }}
      </span>
    </div>
    <div class="_content">
      <p class="_caption section-desc">
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
          <MkButton link to="/admin?tab=users">去创建账号</MkButton>
        </div>
      </AdminCallout>
    </div>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-head h2 {
  font-size: 1em;
}

.section-desc {
  margin-block-end: var(--margin);
}

.count {
  display: inline-grid;
  place-items: center;
  min-inline-size: 22px;
  block-size: 22px;
  padding-inline: 6px;
  border-radius: 999px;
  background: var(--accentedBg);
  color: var(--accent);
  font-size: 0.78em;
  font-weight: bold;
}

.pill {
  padding-block: 2px;
  padding-inline: 10px;
  border: solid 0.5px var(--divider);
  border-radius: 999px;
  color: var(--fgTransparentWeak);
  font-size: 0.78em;
  font-weight: bold;
}

.pill.open {
  border-color: var(--success);
  background: color-mix(in srgb, var(--success) 15%, transparent);
  color: var(--success);
}

.modes {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-block-end: var(--margin);
}

.mode {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: 12px;
  padding-block: 12px;
  padding-inline: 14px;
  border: solid 1px var(--divider);
  border-radius: var(--radius);
  background: var(--buttonBg);
  color: var(--fg);
  font: inherit;
  text-align: start;
  cursor: pointer;
  transition: border-color 0.1s ease, background 0.1s ease, opacity 0.1s ease;
}

.mode:hover:not(:disabled) {
  border-color: var(--accent);
}

.mode:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 1px;
}

.mode.active {
  border-color: var(--accent);
  background: var(--accentedBg);
}

.mode:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.mode-name {
  display: flex;
  align-items: center;
  flex: none;
  gap: 8px;
  min-inline-size: 92px;
  font-weight: bold;
  color: var(--fgHighlighted);
}

.mode-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding-block: 1px;
  padding-inline: 6px 8px;
  border-radius: 999px;
  background: var(--accent);
  color: var(--fgOnAccent);
  font-size: 0.78em;
  font-weight: bold;
}

.mode-tag .icon {
  inline-size: 0.9em;
  block-size: 0.9em;
}

.mode-desc {
  flex: 1;
  min-inline-size: 0;
  font-size: 0.9em;
  color: var(--fgTransparentWeak);
  line-height: 1.4;
}

.notice-row,
.callout-row {
  margin-block-end: var(--margin);
}

.row-form {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  margin-block-end: var(--margin);
}

.row-input {
  flex: 1;
  min-inline-size: 0;
}

.dimmed {
  opacity: 0.55;
}

.hosts {
  list-style: none;
  margin: 0;
  padding: 0;
}

.host {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-block: 8px;
  border-block-start: solid 0.5px var(--hairline);
}

.host:first-child {
  border-block-start: none;
}

.host-name {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.format-hint {
  margin-block-start: 10px;
}

code {
  padding-block: 1px;
  padding-inline: 4px;
  border-radius: 4px;
  background: var(--panelHighlight);
  font-size: 0.88em;
}

@media (max-width: 620px) {
  .mode {
    flex-direction: column;
    gap: 6px;
  }

  .row-form {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
