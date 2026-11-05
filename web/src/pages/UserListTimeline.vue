<!-- Ported from firefish@b425f61:packages/client/src/pages/user-list-timeline.vue (AGPL-3.0-only) -->
<!--
	TODO(wave-3): keep as a placeholder.
	Firefish's list timeline needs `notes/user-list-timeline` plus the `userList`
	streaming channel, and `users/lists/list`. None of them exist in the backend
	(see docs/research/2026-11-05-firefish-port-wave2-contract.md §5 and
	docs/research/api-contract/), so this page renders no data instead of faking a
	timeline. Wire MkTimeline with src="list" once the endpoints land.
-->
<template>
  <div class="cf-page-placeholder">
    <p>TODO: port UserListTimeline</p>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: "CfPageUserListTimeline" });
</script>

<style lang="scss" scoped>
.cf-page-placeholder {
  padding: 32px;
  text-align: center;
  opacity: 0.7;
}
</style>
