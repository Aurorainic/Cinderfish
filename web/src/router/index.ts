import { createRouter, createWebHistory } from "vue-router";

// SPA routes. The Go server reserves /api, /.well-known, /users, /media
// and the other protocol namespaces; everything else falls through to
// index.html. `/notes/:id` and `/@:handle` are served as HTML only when
// the browser asks for text/html, so the same URL still answers
// ActivityPub clients with JSON.
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "home", component: () => import("../views/TimelineView.vue") },
    {
      path: "/login",
      name: "login",
      component: () => import("../views/LoginView.vue"),
      meta: { bare: true },
    },
    {
      path: "/signup",
      name: "signup",
      component: () => import("../views/SignupView.vue"),
      meta: { bare: true },
    },
    {
      path: "/notes/:id",
      name: "note",
      component: () => import("../views/NoteView.vue"),
    },
    {
      path: "/@:handle",
      name: "profile",
      component: () => import("../views/ProfileView.vue"),
    },
    {
      path: "/notifications",
      name: "notifications",
      component: () => import("../views/NotificationsView.vue"),
    },
    {
      path: "/explore",
      name: "explore",
      component: () => import("../views/ExploreView.vue"),
    },
    // Settings is a two-column shell with one child page per tab; the child
    // routes are named `settings-<tab>`, which is how the layout finds the
    // active menu entry. `/settings` lands on the profile tab, and an unknown
    // tab falls back to the same default instead of the 404 page.
    {
      path: "/settings",
      name: "settings",
      component: () => import("../views/SettingsView.vue"),
      children: [
        { path: "", name: "settings-index", redirect: { name: "settings-profile" } },
        {
          path: "profile",
          name: "settings-profile",
          component: () => import("../views/settings/ProfileSettings.vue"),
        },
        {
          path: "appearance",
          name: "settings-appearance",
          component: () => import("../views/settings/AppearanceSettings.vue"),
        },
        {
          path: "navigation",
          name: "settings-navigation",
          component: () => import("../views/settings/NavigationSettings.vue"),
        },
        {
          path: "widgets",
          name: "settings-widgets",
          component: () => import("../views/settings/WidgetSettings.vue"),
        },
        {
          path: "notifications",
          name: "settings-notifications",
          component: () => import("../views/settings/NotificationSettings.vue"),
        },
        {
          path: "privacy",
          name: "settings-privacy",
          component: () => import("../views/settings/PrivacySettings.vue"),
        },
        {
          path: "account",
          name: "settings-account",
          component: () => import("../views/settings/AccountSettings.vue"),
        },
        {
          path: "about",
          name: "settings-about",
          component: () => import("../views/settings/AboutSettings.vue"),
        },
        // A redirect target written as a path instead of a route name keeps the
        // captured `tab` param from being reported as discarded.
        { path: ":tab(.*)", redirect: "/settings/profile" },
      ],
    },
    {
      path: "/admin",
      name: "admin",
      component: () => import("../views/AdminView.vue"),
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("../views/NotFoundView.vue"),
    },
  ],
  scrollBehavior(to, _from, saved) {
    if (saved) return saved;
    if (to.hash) return { el: to.hash, behavior: "smooth" };
    return { top: 0 };
  },
});
