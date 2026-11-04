<template>
  <div class="cf-shell-visitor">
    <header class="topbar">
      <CfA to="/" class="brand">
        <span class="mark">C</span>
        <span class="wordmark">Cinderfish</span>
      </CfA>
      <nav class="links">
        <CfA to="/explore">{{ t("explore") }}</CfA>
        <CfA to="/about">{{ t("about") }}</CfA>
      </nav>
    </header>

    <main class="content">
      <RouterView />
    </main>

    <footer class="footer">
      <span>Cinderfish</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
// Ported from Firefish's ui/visitor.vue: a centred, chrome-light layout for
// logged-out visitors (welcome, about, explore).
import { RouterView } from "vue-router";
import CfA from "../components/ui/CfA.vue";
import { useLocale } from "../i18n";

const { t } = useLocale();
</script>

<style lang="scss" scoped>
.cf-shell-visitor {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  max-width: 1100px;
  width: 100%;
  margin: 0 auto;
  padding: 16px var(--margin);
  box-sizing: border-box;

  > .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 800;

    > .mark {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 30px;
      height: 30px;
      border-radius: 10px;
      color: var(--fgOnAccent);
      background: linear-gradient(135deg, var(--accent), var(--renote));
    }
  }

  > .links {
    display: flex;
    gap: 16px;
    font-size: 0.9em;
  }
}

.content {
  flex: 1;
  width: 100%;
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 var(--margin) var(--margin);
  box-sizing: border-box;
}

.footer {
  padding: 24px;
  text-align: center;
  font-size: 0.8em;
  color: var(--fgTransparentWeak);
}
</style>
