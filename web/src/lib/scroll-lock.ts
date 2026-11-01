// One owner for the body scroll lock.
//
// The nav drawer and the widgets drawer can be open at the same time, and each
// used to write `document.body.style.overflow` on its own: closing them in the
// order they were opened restored an empty string over the other one's lock
// (the page stayed frozen) or unlocked while a panel was still open. A counter
// makes the lock a shared resource and restores the value found before the
// first lock.
let locks = 0;
let saved = "";

/** Lock page scrolling. Returns an idempotent release function. */
export function lockBodyScroll(): () => void {
  if (locks === 0) {
    saved = document.body.style.overflow;
    document.body.style.overflow = "hidden";
  }
  locks += 1;
  let released = false;
  return () => {
    if (released) return;
    released = true;
    locks = Math.max(0, locks - 1);
    if (locks === 0) document.body.style.overflow = saved;
  };
}
