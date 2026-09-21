(function () {
  "use strict";

  var reduceMotion = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  function initActionFeedback() {
    document.addEventListener("submit", function (event) {
      var submitter = event.submitter;
      if (!submitter || !submitter.classList || !submitter.classList.contains("btn-base")) {
        return;
      }

      submitter.classList.add("is-pressed");
      window.setTimeout(function () {
        submitter.classList.remove("is-pressed");
      }, 220);
    });
  }

  function initReveal() {
    if (reduceMotion || !("IntersectionObserver" in window)) {
      return;
    }

    var selectors = [
      ".hero-grid > div",
      ".section-heading",
      ".dna-card",
      ".process-step",
      ".catalog-layout > div",
      ".catalog-page-hero-grid > div",
      ".catalog-toolbar",
      ".product-card",
      ".product-detail-visual",
      ".product-detail-content",
      ".cart-heading",
      ".cart-line",
      ".cart-summary",
      ".checkout-heading",
      ".checkout-step",
      ".checkout-form-section",
      ".checkout-summary",
      ".empty-state",
      ".cta-layout",
      ".lab-story-grid > div",
      ".brand-impact-title",
      ".admin-metric",
      ".admin-panel",
      ".admin-order-row",
      ".admin-item"
    ];

    var elements = Array.prototype.slice.call(document.querySelectorAll(selectors.join(",")));
    if (elements.length === 0) {
      return;
    }

    elements.forEach(function (element, index) {
      element.classList.add("motion-reveal");
      element.style.setProperty("--reveal-delay", Math.min(index % 6, 5) * 45 + "ms");
    });

    document.documentElement.classList.add("motion-ready");

    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) {
          return;
        }

        entry.target.classList.add("motion-visible");
        observer.unobserve(entry.target);
      });
    }, {
      rootMargin: "0px 0px -10% 0px",
      threshold: 0.08
    });

    elements.forEach(function (element) {
      observer.observe(element);
    });
  }

  function init() {
    initActionFeedback();
    initReveal();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
    return;
  }

  init();
})();
