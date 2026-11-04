<template>
  <Timeline v-if="me != null" />
  <Welcome v-else />
</template>

<script setup lang="ts">
// The "/" route. Firefish chooses timeline.vue vs welcome.vue at module load
// (`$i ? timeline : welcome`); this keeps the choice reactive so signing in
// without a reload swaps the page.
import { me } from "../account";
import Timeline from "./Timeline.vue";
import Welcome from "./Welcome.vue";

defineOptions({ name: "CfPageIndexRoute" });
</script>
