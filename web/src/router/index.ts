import { createRouter, createWebHistory } from "vue-router";
import { refreshSession, session } from "../stores/session";

// SPA routes. The Go server reserves /api, /.well-known, /users, /media
// and the other protocol namespaces; everything else falls through to
// index.html. `/notes/:id` and `/@:handle` are served as HTML only when
// the browser asks for text/html, so the same URL still answers
// ActivityPub clients with JSON.
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    // `meta.deck` marks the surfaces the deck can render as columns. A route
    // without it stays in the universal layout even when the deck preference is
    // on, which is what keeps deep links to notes, profiles and settings
    // scrollable.
    {
      path: "/",
      name: "home",
      component: () => import("../views/TimelineView.vue"),
      meta: { deck: true },
    },
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
      meta: { deck: true },
    },
    {
      path: "/explore",
      name: "explore",
      component: () => import("../views/ExploreView.vue"),
      meta: { deck: true },
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
          meta: { requiresAuth: true },
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
          meta: { requiresAuth: true },
        },
        {
          path: "privacy",
          name: "settings-privacy",
          component: () => import("../views/settings/PrivacySettings.vue"),
          meta: { requiresAuth: true },
        },
        {
          path: "account",
          name: "settings-account",
          component: () => import("../views/settings/AccountSettings.vue"),
          meta: { requiresAuth: true },
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
      meta: { requiresAuth: true },
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

// Account-only routes gate on the resolved session store, never on the raw
// token: the guard awaits the shared refresh (it returns immediately when no
// token is stored) so a deep link into a gated page either lands once the
// account is loaded or sends the guest to /login with the original location
// kept in `redirect`, which LoginView consumes after a successful signin.
// Public routes — timeline, notes, profiles, explore, login, signup and the
// client-only appearance/navigation/widgets preferences — are never gated.
// Authorization itself stays server-side; this guard is UX only.
router.beforeEach(async (to) => {
  if (!to.meta.requiresAuth) return true;
  if (!session.ready) await refreshSession();
  if (session.me) return true;
  return { path: "/login", query: { redirect: to.fullPath } };
});
