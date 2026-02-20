(() => {
  if (typeof document$ === "undefined") {
    return;
  }

  let firstRender = true;

  document$.subscribe(() => {
    if (firstRender) {
      firstRender = false;
      return;
    }

    if (window.ethicalads && typeof window.ethicalads.reload === "function") {
      window.ethicalads.reload();
    }
  });
})();
