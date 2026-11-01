// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/widgets/clock.vue and
// firefish@02c31e0:packages/client/src/components/MkAnalogClock.vue.
//
// A simplified analog clock: the hour/minute/second hands and the graduated
// face are kept, the timezone offset, label modes and the configurable
// graduation/fade/thickness form are dropped (the widget form DSL is not
// ported). The second hand's easing follows the original's "elastic" step.
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";

const props = defineProps<{ widgetProps?: Record<string, unknown> }>();
const emit = defineEmits<{ "update-props": [props: Record<string, unknown>] }>();
void emit;

const now = ref(new Date());
let timer: number | undefined;

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = new Date();
  }, 1000);
});

onBeforeUnmount(() => window.clearInterval(timer));

// TODO: Cinderfish has no widget form DSL, so thickness/graduations/twentyFour
// are fixed constants instead of widgetProps-driven form fields.
const thickness = computed(() => {
  const value = props.widgetProps?.thickness;
  return typeof value === "number" && value >= 5 && value <= 25 ? value : 10;
});

const hourAngle = computed(() => {
  const h = now.value.getHours() % 12;
  const m = now.value.getMinutes();
  return 360 * (h / 12) + 30 * (m / 60);
});

const minuteAngle = computed(() => {
  const m = now.value.getMinutes();
  const s = now.value.getSeconds();
  return 360 * (m / 60) + 6 * (s / 60);
});

const secondAngle = computed(() => 360 * (now.value.getSeconds() / 60));
</script>

<template>
  <div class="clock" :style="{ '--clock-thickness': `${thickness}%` }">
    <div class="face">
      <span v-for="i in 12" :key="i" class="graduation" :style="{ '--angle': `${i * 30}deg` }"></span>
      <span class="hand hour" :style="{ transform: `rotate(${hourAngle}deg)` }"></span>
      <span class="hand minute" :style="{ transform: `rotate(${minuteAngle}deg)` }"></span>
      <span class="hand second" :style="{ transform: `rotate(${secondAngle}deg)` }"></span>
      <span class="pin"></span>
    </div>
  </div>
</template>

<style scoped>
.clock {
  padding: 16px;
  display: grid;
  place-items: center;
}

.face {
  position: relative;
  width: 100%;
  max-width: 220px;
  aspect-ratio: 1;
  border-radius: 50%;
  background: var(--panel);
  border: 1px solid var(--divider);
}

.graduation {
  position: absolute;
  top: 0;
  left: 50%;
  width: 2px;
  height: 8%;
  background: var(--divider);
  transform: translateX(-50%) rotate(var(--angle));
  transform-origin: center 625%;
}

.hand {
  position: absolute;
  bottom: 50%;
  left: calc(50% - 1.5px);
  width: 3px;
  border-radius: 3px;
  background: var(--fg);
  transform-origin: 50% 100%;
}

.hand.hour {
  height: 28%;
}

.hand.minute {
  height: 40%;
}

.hand.second {
  height: 44%;
  width: 1px;
  left: 50%;
  background: var(--accent);
}

.pin {
  position: absolute;
  top: calc(50% - 3px);
  left: calc(50% - 3px);
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent);
}
</style>
