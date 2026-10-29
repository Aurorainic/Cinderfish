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
    {
      path: "/settings",
      name: "settings",
      component: () => import("../views/SettingsView.vue"),
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
