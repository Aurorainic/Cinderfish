// Route table, translated from Firefish's src/router.ts (nirax).
//
// Notes on the translation:
//   * nirax children match against the remaining path, so Firefish's children
//     written as "/profile" under "/settings" resolve to "/settings/profile".
//     vue-router needs the absolute child path, which is what is written here.
//   * Firefish has duplicate route names (three "theme", several "profile",
//     two "general", ...). vue-router warns on duplicates, so each child gets
//     a unique name or none at all; paths are preserved.
//   * duplicate paths in the original table (/custom-css twice under settings,
//     /account-info twice, ...) are deduplicated; the first entry wins.
//   * every component is a placeholder page until the page-port batch lands.
import { defineAsyncComponent, type Component } from "vue";
import type { RouteRecordRaw } from "vue-router";
import CfInfo from "../components/ui/CfInfo.vue";
import CfLoading from "../components/ui/CfLoading.vue";

function page(name: string) {
  return defineAsyncComponent({
    loader: () => import(`../pages/${name}.vue`) as Promise<{ default: Component }>,
    loadingComponent: CfLoading,
    errorComponent: CfInfo,
    delay: 0,
  });
}

const login = { requiresLogin: true } as const;

export const routes: RouteRecordRaw[] = [
  // ----- user pages ---------------------------------------------------------
  { path: "/@:initUser/pages/:initPageName/view-source", component: page("PageEditor") },
  { path: "/@:username/pages/:pageName", component: page("Page") },
  { path: "/@:acct/following", component: page("UserFollowing") },
  { path: "/@:acct/followers", component: page("UserFollowers") },
  { path: "/@:acct/:page?", name: "user", component: page("UserProfile") },
  { path: "/notes/:noteId", name: "note", component: page("Note") },
  { path: "/notes/:noteId/edit", name: "note-edit", component: page("NoteEdit") },
  { path: "/notes/:noteId/history", name: "note-history", component: page("NoteHistory") },
  { path: "/clips/:clipId", component: page("Clip") },
  { path: "/user-info/:userId", component: page("UserInfo") },
  { path: "/instance-info/:host", component: page("InstanceInfo") },
  { path: "/public/local", component: page("NoGraze") },

  // ----- settings -----------------------------------------------------------
  {
    path: "/settings",
    name: "settings",
    component: page("SettingsIndex"),
    meta: login,
    children: [
      { path: "/settings/profile", name: "settings-profile", component: page("SettingsProfile") },
      { path: "/settings/privacy", name: "settings-privacy", component: page("SettingsPrivacy") },
      { path: "/settings/reaction", name: "settings-reaction", component: page("SettingsReaction") },
      { path: "/settings/accessibility", name: "settings-accessibility", component: page("SettingsAccessibility") },
      { path: "/settings/drive", name: "settings-drive", component: page("SettingsDrive") },
      { path: "/settings/notifications", name: "settings-notifications", component: page("SettingsNotifications") },
      { path: "/settings/email", name: "settings-email", component: page("SettingsEmail") },
      { path: "/settings/integration", name: "settings-integration", component: page("SettingsIntegration") },
      { path: "/settings/security", name: "settings-security", component: page("SettingsSecurity") },
      { path: "/settings/general", name: "settings-general", component: page("SettingsGeneral") },
      { path: "/settings/theme/install", name: "settings-theme-install", component: page("SettingsThemeInstall") },
      { path: "/settings/theme/manage", name: "settings-theme-manage", component: page("SettingsThemeManage") },
      { path: "/settings/theme", name: "settings-theme", component: page("SettingsTheme") },
      { path: "/settings/custom-css", name: "settings-custom-css", component: page("SettingsCustomCss") },
      {
        path: "/settings/custom-katex-macro",
        name: "settings-custom-katex-macro",
        component: page("SettingsCustomKatexMacro"),
      },
      { path: "/settings/account-info", name: "settings-account-info", component: page("SettingsAccountInfo") },
      { path: "/settings/navbar", name: "settings-navbar", component: page("SettingsNavbar") },
      { path: "/settings/statusbar", name: "settings-statusbar", component: page("SettingsStatusbar") },
      { path: "/settings/sounds", name: "settings-sounds", component: page("SettingsSounds") },
      { path: "/settings/plugin/install", name: "settings-plugin-install", component: page("SettingsPluginInstall") },
      { path: "/settings/plugin", name: "settings-plugin", component: page("SettingsPlugin") },
      { path: "/settings/import-export", name: "settings-import-export", component: page("SettingsImportExport") },
      { path: "/settings/instance-mute", name: "settings-instance-mute", component: page("SettingsInstanceMute") },
      { path: "/settings/mute-block", name: "settings-mute-block", component: page("SettingsMuteBlock") },
      { path: "/settings/word-mute", name: "settings-word-mute", component: page("SettingsWordMute") },
      { path: "/settings/api", name: "settings-api", component: page("SettingsApi") },
      { path: "/settings/apps", name: "settings-apps", component: page("SettingsApps") },
      {
        path: "/settings/webhook/edit/:webhookId",
        name: "settings-webhook-edit",
        component: page("SettingsWebhookEdit"),
      },
      { path: "/settings/webhook/new", name: "settings-webhook-new", component: page("SettingsWebhookNew") },
      { path: "/settings/webhook", name: "settings-webhook", component: page("SettingsWebhook") },
      { path: "/settings/deck", name: "settings-deck", component: page("SettingsDeck") },
      {
        path: "/settings/delete-account",
        name: "settings-delete-account",
        component: page("SettingsDeleteAccount"),
      },
      {
        path: "/settings/preferences-backups",
        name: "settings-preferences-backups",
        component: page("SettingsPreferencesBackups"),
      },
      { path: "/settings/migration", name: "settings-migration", component: page("SettingsMigration") },
      { path: "/settings/accounts", name: "settings-accounts", component: page("SettingsAccounts") },
      { path: "/settings/other", name: "settings-other", component: page("SettingsOther") },
      { path: "", name: "settings-index", component: page("Empty") },
    ],
  },

  // ----- auth flows ---------------------------------------------------------
  { path: "/reset-password/:token?", component: page("ResetPassword") },
  { path: "/signup-complete/:code", component: page("SignupComplete") },
  { path: "/verify-email/:code", component: page("VerifyEmail") },
  { path: "/auth/:token", component: page("Auth") },
  { path: "/oauth/authorize", name: "oauth", component: page("OAuth") },
  { path: "/miauth/:session", component: page("MiAuth") },

  // ----- static pages -------------------------------------------------------
  { path: "/announcements", component: page("Announcements") },
  { path: "/about", component: page("About") },
  { path: "/about-firefish", component: page("AboutFirefish") },
  { path: "/theme-editor", component: page("ThemeEditor"), meta: login },
  { path: "/explore/tags/:tag", component: page("Explore") },
  { path: "/explore", component: page("Explore") },
  { path: "/search", component: page("Search") },
  { path: "/authorize-follow", component: page("Follow"), meta: login },
  { path: "/follow-me", component: page("FollowMe") },
  { path: "/authorize_interaction", component: page("AuthorizeInteraction"), meta: login },
  { path: "/share", component: page("Share"), meta: login },
  { path: "/api-console", component: page("ApiConsole"), meta: login },
  { path: "/mfm-cheat-sheet", component: page("MfmCheatSheet") },
  { path: "/scratchpad", component: page("Scratchpad") },
  { path: "/preview", component: page("Preview") },
  { path: "/tags/:tag", component: page("Tag") },

  // ----- pages / gallery / channels ----------------------------------------
  { path: "/pages/new", component: page("PageEditor"), meta: login },
  { path: "/pages/edit/:initPageId", component: page("PageEditor"), meta: login },
  { path: "/pages", component: page("Pages") },
  { path: "/gallery/:postId/edit", component: page("GalleryEdit"), meta: login },
  { path: "/gallery/new", component: page("GalleryEdit"), meta: login },
  { path: "/gallery/:postId", component: page("GalleryPost") },
  { path: "/gallery", component: page("GalleryIndex") },
  { path: "/channels/:channelId/edit", component: page("ChannelEditor"), meta: login },
  { path: "/channels/new", component: page("ChannelEditor"), meta: login },
  { path: "/channels/:channelId", component: page("Channel") },
  { path: "/channels", component: page("Channels") },

  // ----- registry -----------------------------------------------------------
  { path: "/registry/keys/system/:path(.*)*", component: page("RegistryKeys") },
  { path: "/registry/value/system/:path(.*)*", component: page("RegistryValue") },
  { path: "/registry", component: page("Registry") },

  // ----- administration -----------------------------------------------------
  { path: "/admin/file/:fileId", component: page("AdminFile"), meta: login },
  {
    path: "/admin",
    component: page("AdminIndex"),
    meta: login,
    children: [
      { path: "/admin/overview", name: "admin-overview", component: page("AdminOverview") },
      { path: "/admin/users", name: "admin-users", component: page("AdminUsers") },
      { path: "/admin/hashtags", name: "admin-hashtags", component: page("AdminHashtags") },
      { path: "/admin/emojis", name: "admin-emojis", component: page("AdminEmojis") },
      { path: "/admin/federation", name: "admin-federation", component: page("AdminFederation") },
      { path: "/admin/queue", name: "admin-queue", component: page("AdminQueue") },
      { path: "/admin/files", name: "admin-files", component: page("AdminFiles") },
      { path: "/admin/announcements", name: "admin-announcements", component: page("AdminAnnouncements") },
      { path: "/admin/ads", name: "admin-ads", component: page("AdminPromotions") },
      { path: "/admin/database", name: "admin-database", component: page("AdminDatabase") },
      { path: "/admin/abuses", name: "admin-abuses", component: page("AdminAbuses") },
      { path: "/admin/settings", name: "admin-settings", component: page("AdminSettings") },
      { path: "/admin/email-settings", name: "admin-email-settings", component: page("AdminEmailSettings") },
      { path: "/admin/custom-css", name: "admin-custom-css", component: page("AdminCustomCss") },
      {
        path: "/admin/object-storage",
        name: "admin-object-storage",
        component: page("AdminObjectStorage"),
      },
      { path: "/admin/security", name: "admin-security", component: page("AdminSecurity") },
      { path: "/admin/relays", name: "admin-relays", component: page("AdminRelays") },
      { path: "/admin/integrations", name: "admin-integrations", component: page("AdminIntegrations") },
      { path: "/admin/instance-block", name: "admin-instance-block", component: page("AdminInstanceBlock") },
      { path: "/admin/proxy-account", name: "admin-proxy-account", component: page("AdminProxyAccount") },
      {
        path: "/admin/other-settings",
        name: "admin-other-settings",
        component: page("AdminOtherSettings"),
      },
      { path: "/admin/experiments", name: "admin-experiments", component: page("AdminExperiments") },
      { path: "", name: "admin-index", component: page("Empty") },
    ],
  },

  // ----- my/* ---------------------------------------------------------------
  { path: "/my/notifications", component: page("Notifications"), meta: login },
  { path: "/my/favorites", component: page("Favorites"), meta: login },
  { path: "/my/messaging", name: "messaging", component: page("MessagingIndex"), meta: login },
  { path: "/my/messaging/:userAcct", component: page("MessagingRoom"), meta: login },
  { path: "/my/messaging/group/:groupId", component: page("MessagingRoom"), meta: login },
  { path: "/my/drive/folder/:folder", component: page("Drive"), meta: login },
  { path: "/my/drive/file/:fileId/attached", component: page("AttachedFiles"), meta: login },
  { path: "/my/drive", component: page("Drive"), meta: login },
  { path: "/my/follow-requests", component: page("FollowRequests"), meta: login },
  { path: "/my/follow-requests/sent", component: page("FollowRequestsSent"), meta: login },
  { path: "/my/lists/:listId", component: page("MyListsList"), meta: login },
  { path: "/my/lists", component: page("MyListsIndex"), meta: login },
  { path: "/my/clips", component: page("MyClips"), meta: login },
  { path: "/my/groups", component: page("MyGroupsIndex"), meta: login },
  { path: "/my/groups/:groupId", component: page("MyGroupsGroup"), meta: login },
  { path: "/my/antennas/create", component: page("MyAntennasCreate"), meta: login },
  { path: "/my/antennas/:antennaId", component: page("MyAntennasEdit"), meta: login },
  { path: "/my/antennas", component: page("MyAntennasIndex"), meta: login },
  { path: "/timeline/list/:listId", component: page("UserListTimeline"), meta: login },
  { path: "/timeline/antenna/:antennaId", component: page("AntennaTimeline"), meta: login },

  // ----- index / deck -------------------------------------------------------
  // Firefish picks timeline vs welcome at module load; IndexRoute.vue does it
  // reactively instead.
  { path: "/", name: "index", component: page("IndexRoute"), meta: { shell: "home" } },
  // Additions requested by the port brief: /timeline keeps the Firefish path
  // (IndexRoute shows the signed-in timeline or the welcome page), and /deck
  // forces the deck shell for the same index content.
  { path: "/timeline", name: "timeline", component: page("IndexRoute"), meta: { shell: "home" } },
  { path: "/deck", name: "deck", component: page("IndexRoute"), meta: { shell: "deck" } },

  { path: "/:pathMatch(.*)*", component: page("NotFound") },
];
