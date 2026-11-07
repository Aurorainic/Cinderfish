<!-- Ported from firefish@b425f61:packages/client/src/pages/user/home.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="narrow ? 800 : 1100">
		<div ref="rootEl" class="ftskorzw" :class="{ wide: !narrow }">
			<div class="main">
				<div class="profile">
					<MkMoved
						v-if="user.movedToUri"
						class="moved"
						:host="user.movedToUri.host ?? ''"
						:acct="user.movedToUri.username"
					/>
					<div :key="user.id" class="card _block">
						<div class="banner-container">
							<div class="banner" :style="bannerStyle"></div>
							<div class="fade"></div>
							<div class="title">
								<div class="nameColumn">
									<MkUserName class="name" :user="user" :nowrap="true" />
									<div class="badges">
										<span
											v-if="user.isAdmin"
											:title="t('isAdmin')"
											class="badge"
											><i class="ph-bookmark-simple ph-fill ph-lg"></i
										></span>
										<span
											v-else-if="user.isModerator"
											:title="t('isModerator')"
											class="badge"
											><i class="ph-bookmark-simple ph-bold ph-lg"></i
										></span>
										<span v-if="user.isLocked" :title="t('isLocked')"
											><i class="ph-lock ph-bold ph-lg"></i
										></span>
										<span v-if="user.isBot" :title="t('isBot')"
											><i class="ph-robot ph-bold ph-lg"></i
										></span>
										<span v-if="user.isSuspended" class="suspended">
											<i class="ph-warning ph-bold ph-lg"></i>
											{{ t("suspend") }}
										</span>
									</div>
									<span v-if="followsYou" class="followed">{{
										t("followsYou")
									}}</span>
								</div>
								<div class="bottom">
									<span class="username"
										><MkAcct :user="user" :detail="true"
									/></span>
								</div>
							</div>
						</div>
						<MkAvatar
							class="avatar"
							:user="user"
							:disable-preview="true"
							:show-indicator="true"
						/>
						<div class="actions">
							<MkFollowButton
								:user="user"
								class="koudoku"
								:full="true"
								@refresh="refresh"
							/>
							<button
								class="_buttonIcon menu"
								:title="t('menu')"
								@click.stop="openMenu"
							>
								<i class="ph-dots-three ph-bold ph-lg"></i>
							</button>
						</div>
						<!-- Firefish keeps one title on the banner for wide layouts and a
						     second one below the avatar for mobile; both render the same
						     data so the CSS can pick the right one. -->
						<div class="title mobile-title">
							<div class="nameColumn">
								<MkUserName class="name" :user="user" :nowrap="true" />
								<div class="badges">
									<span
										v-if="user.isAdmin"
										:title="t('isAdmin')"
										class="badge"
										><i class="ph-bookmark-simple ph-fill ph-lg"></i
									></span>
									<span
										v-else-if="user.isModerator"
										:title="t('isModerator')"
										class="badge"
										><i class="ph-bookmark-simple ph-bold ph-lg"></i
									></span>
									<span v-if="user.isLocked" :title="t('isLocked')"
										><i class="ph-lock ph-bold ph-lg"></i
									></span>
									<span v-if="user.isBot" :title="t('isBot')"
										><i class="ph-robot ph-bold ph-lg"></i
									></span>
									<span v-if="user.isSuspended" class="suspended">
										<i class="ph-warning ph-bold ph-lg"></i>
										{{ t("suspend") }}
									</span>
								</div>
								<span v-if="followsYou" class="followed">{{
									t("followsYou")
								}}</span>
							</div>
							<div class="bottom">
								<span class="username"
									><MkAcct :user="user" :detail="true"
								/></span>
							</div>
						</div>
						<div v-if="user.host != null" class="remote">
							<CfInfo warn>{{ t("remoteUserCaution") }}</CfInfo>
						</div>
						<div class="description">
							<Mfm
								v-if="user.description"
								:text="user.description"
								:is-note="false"
								:author="user"
								:i="me"
								:custom-emojis="emojis"
							/>
							<p v-else class="empty">{{ t("noAccountDescription") }}</p>
						</div>
						<div class="fields system">
							<dl class="field">
								<dt class="name">
									<i class="ph-calendar-blank ph-bold ph-lg ph-fw"></i>
									{{ t("registeredDate") }}
								</dt>
								<dd class="value">
									<MkTime :time="user.createdAt" mode="detail" />
								</dd>
							</dl>
						</div>
						<div v-if="fields.length > 0" class="fields">
							<dl v-for="(field, i) in fields" :key="i" class="field">
								<dt class="name">
									<Mfm
										:text="field.name"
										:plain="true"
										:custom-emojis="emojis"
									/>
								</dt>
								<dd class="value">
									<Mfm
										:text="field.value"
										:author="user"
										:i="me"
										:custom-emojis="emojis"
									/>
								</dd>
							</dl>
						</div>
						<div class="status">
							<MkA :to="userPage(user)">
								<b><MkNumber :value="user.notesCount" /></b>
								<span>{{ t("notes") }}</span>
							</MkA>
							<MkA :to="userPage(user, 'following')">
								<b><MkNumber :value="user.followingCount" /></b>
								<span>{{ t("following") }}</span>
							</MkA>
							<MkA :to="userPage(user, 'followers')">
								<b><MkNumber :value="user.followersCount" /></b>
								<span>{{ t("followers") }}</span>
							</MkA>
						</div>
					</div>
				</div>

				<div class="contents _gap">
					<div v-if="user.pinnedNotes.length > 0" class="_gap">
						<MkNote
							v-for="note in user.pinnedNotes"
							:key="note.id"
							class="pinned _block"
							:note="note"
							:pinned="true"
						/>
					</div>
					<CfInfo v-else-if="isMe" class="pinTip">{{
						t("userPagePinTip")
					}}</CfInfo>
					<template v-if="narrow">
						<UserPhotos :key="user.id" :user="user" />
					</template>
				</div>
				<div>
					<UserNotes :user="user" :with-replies="false" />
				</div>
			</div>
			<div v-if="!narrow" class="sub">
				<UserPhotos :key="user.id" :user="user" />
			</div>
		</div>
	</MkSpacer>
</template>

<script lang="ts" setup>
// Degradations versus Firefish home.vue (contract §2 and the API contract):
//   * no scroll parallax (the banner is a static background);
//   * no location / birthday rows and no "their local time": both fields are
//     always null and `city-timezones` is not a dependency;
//   * no `patrons` badge: the endpoint does not exist;
//   * no remote-caution link target: `url` is an empty string, so the caution
//     is a plain CfInfo;
//   * the sidebar chart is dropped; it needs `charts/user/notes`.
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import type { EmojiSimple, UserDetailed } from "@/api/types";
import { me } from "@/account";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkFollowButton from "@/components/MkFollowButton.vue";
import MkMoved from "@/components/MkMoved.vue";
import MkNote from "@/components/MkNote.vue";
import MkNumber from "@/components/MkNumber.vue";
import UserNotes from "@/components/user/UserNotes.vue";
import UserPhotos from "@/components/user/UserPhotos.vue";
import { userPage } from "@/filters/user";
import { t } from "@/i18n";
import { getUserMenu } from "@/scripts/get-user-menu";

interface ProfileField {
	name: string;
	value: string;
}

const props = defineProps<{
	user: UserDetailed;
}>();

const emit = defineEmits<{
	refresh: [];
}>();

const bannerStyle = computed(() =>
	props.user.bannerUrl == null || props.user.bannerUrl === ""
		? undefined
		: {
				backgroundImage: `url(${props.user.bannerUrl})`,
				backgroundSize: "cover",
				backgroundPosition: "center",
			},
);

const rootEl = ref<HTMLElement | null>(null);
const narrow = ref(window.innerWidth < 1000);

const isMe = computed<boolean>(() => me.value?.id === props.user.id);

const followsYou = computed<boolean>(
	() => me.value != null && me.value.id !== props.user.id && props.user.isFollowed,
);

const emojis = computed<EmojiSimple[]>(
	() => (props.user.emojis ?? []) as unknown as EmojiSimple[],
);

const fields = computed<ProfileField[]>(
	() => (props.user.fields ?? []) as ProfileField[],
);

function refresh(): void {
	emit("refresh");
}

function openMenu(ev: MouseEvent): void {
	getUserMenu(props.user, ev);
}

let ro: ResizeObserver | null = null;

onMounted(() => {
	ro = new ResizeObserver(() => {
		if (rootEl.value == null) return;
		narrow.value = rootEl.value.clientWidth < 1000;
	});
	if (rootEl.value != null) ro.observe(rootEl.value);
});

onBeforeUnmount(() => {
	ro?.disconnect();
	ro = null;
});
</script>

<style lang="scss" scoped>
.ftskorzw {
	> .main {
		min-width: 0;
		flex: 1;

		> .profile > .card {
			position: relative;
			overflow: hidden;

			> .banner-container {
				position: relative;
				height: 250px;
				overflow: hidden;

				> .banner {
					height: 100%;
					background: linear-gradient(120deg, var(--accentedBg), var(--buttonBg));
				}

				> .fade {
					position: absolute;
					bottom: 0;
					left: 0;
					width: 100%;
					height: 78px;
					background: linear-gradient(transparent, rgb(0 0 0 / 70%));
				}
			}

			> .avatar {
				display: block;
				position: absolute;
				top: 170px;
				left: 16px;
				z-index: 2;
				width: 120px;
				height: 120px;
				box-shadow: 1px 1px 3px rgb(0 0 0 / 20%);
			}

			> .banner-container > .title {
				position: absolute;
				bottom: 0;
				left: 0;
				width: 100%;
				padding: 0 0 8px 154px;
				box-sizing: border-box;
				color: #fff;

				> .nameColumn {
					display: block;

					> .name {
						margin: 0;
						line-height: 32px;
						font-weight: bold;
						font-size: 1.8em;
						text-shadow: 0 0 8px rgb(0 0 0 / 60%);
					}

					> .badges {
						display: inline-flex;
						align-items: center;
						gap: 4px;
						vertical-align: middle;
						opacity: 0.9;
					}

					> .followed {
						position: relative;
						top: -4px;
						left: 4px;
						padding: 4px 8px;
						color: #fff;
						background: rgb(0 0 0 / 60%);
						font-size: 0.7em;
						border-radius: 24px;
					}

					> .suspended {
						font-size: 0.7em;
					}
				}

				> .bottom > * {
					display: inline-block;
					margin-right: 16px;
					line-height: 20px;
					opacity: 0.8;

					&.username {
						font-weight: bold;
					}
				}
			}

			> .actions {
				position: absolute;
				top: 200px;
				right: 12px;
				z-index: 3;
				display: flex;
				align-items: center;
				gap: 8px;
			}

			> .title.mobile-title {
				display: none;
			}

			> .remote {
				padding: 64px 16px 0 16px;
			}

			> .description {
				padding: 72px 12px 8px 24px;
				font-size: 0.95em;

				> .empty {
					margin: 0;
					opacity: 0.5;
				}
			}

			> .fields {
				padding: 24px;
				font-size: 0.9em;
				border-top: solid 0.5px var(--divider);

				> .field {
					display: flex;
					padding: 0;
					margin: 0;
					align-items: center;

					&:not(:last-child) {
						margin-bottom: 8px;
					}

					> .name {
						width: 30%;
						overflow: hidden;
						white-space: nowrap;
						text-overflow: ellipsis;
						font-weight: bold;
						text-align: center;
					}

					> .value {
						width: 70%;
						overflow: hidden;
						white-space: nowrap;
						text-overflow: ellipsis;
						margin: 0;
					}
				}
			}

			> .status {
				display: flex;
				padding: 24px;
				border-top: solid 0.5px var(--divider);

				> a {
					flex: 1;
					text-align: center;

					&:hover {
						text-decoration: none;
					}

					> b {
						display: block;
						line-height: 16px;
					}

					> span {
						font-size: 70%;
					}
				}
			}
		}

		> .contents > .pinTip {
			margin: 12px 0;
		}
	}

	&.wide {
		display: flex;
		width: 100%;

		> .sub {
			max-width: 350px;
			min-width: 350px;
			margin-left: var(--margin);
		}
	}

	@media (max-width: 500px) {
		> .main > .profile > .card {
			> .banner-container {
				height: 140px;

				> .fade {
					display: none;
				}

				> .title {
					display: none;
				}
			}

			> .title.mobile-title {
				display: block;
				position: static;
				padding: 8px 16px 12px 16px;
				color: var(--fg);
				text-align: center;

				> .nameColumn > .name {
					font-size: 1.3em;
					text-shadow: none;
				}
			}

			> .avatar {
				top: 92px;
				left: 0;
				right: 0;
				width: 92px;
				height: 92px;
				margin: auto;
			}

			> .actions {
				position: static;
				justify-content: center;
				padding: 52px 8px 8px;
			}

			> .description,
			> .fields,
			> .status {
				padding: 16px;
			}

			> .description {
				padding-top: 16px;
				top: 0;
			}
		}
	}
}
</style>
