import { createRouter, createWebHistory, type Router } from "vue-router";
import { authGuard } from "../account";
import { routes } from "./routes";

export const router: Router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition != null) return savedPosition;
    if (to.hash.length > 0) return { el: to.hash, behavior: "smooth" };
    return { top: 0 };
  },
});

// Firefish's nirax router consulted `loginRequired` per route and called
// pleaseLogin(); here routes carry `meta.requiresLogin` and the account
// layer's authGuard redirects to the welcome page with a `redirect` query.
router.beforeEach(authGuard);

export { router as mainRouter, routes };
export { RouterLink, RouterView, useRoute, useRouter } from "vue-router";
