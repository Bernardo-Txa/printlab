(function () {
  "use strict";

  var reduceMotion = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  function initActiveNavigation() {
    var links = Array.prototype.slice.call(document.querySelectorAll(".site-nav .nav-link, .site-header-actions .nav-link"));
    if (links.length === 0) {
      return;
    }

    var path = window.location.pathname || "/";
    var hash = window.location.hash || "";
    links.forEach(function (link) {
      var href = link.getAttribute("href") || "";
      var active = false;
      if (path === "/" && href === "/#inicio" && (!hash || hash === "#inicio")) {
        active = true;
      } else if (path === "/" && hash && href === "/" + hash) {
        active = true;
      } else if (href === path) {
        active = true;
      } else if (path.indexOf("/produtos") === 0 && href === "/produtos") {
        active = true;
      } else if (path.indexOf("/carrinho") === 0 && href === "/carrinho") {
        active = true;
      } else if (path.indexOf("/conta") === 0 && href === "/conta") {
        active = true;
      }

      if (active) {
        link.classList.add("is-active");
        link.setAttribute("aria-current", path === "/" && href.indexOf("#") > -1 ? "location" : "page");
      }
    });
  }

  function initActionFeedback() {
    document.addEventListener("submit", function (event) {
      var submitter = event.submitter;
      if (!submitter || !submitter.classList) {
        return;
      }

      if (submitter.classList.contains("btn-base") || submitter.classList.contains("cart-remove-button")) {
        submitter.classList.add("is-pressed", "is-submitting");
        submitter.setAttribute("aria-busy", "true");
        window.setTimeout(function () {
          submitter.classList.remove("is-pressed");
        }, 220);
        window.setTimeout(function () {
          submitter.classList.remove("is-submitting");
          submitter.removeAttribute("aria-busy");
        }, 1400);
      }

      var form = submitter && submitter.form;
      if (form && form.classList) {
        form.classList.add("is-submitting");
        window.setTimeout(function () {
          form.classList.remove("is-submitting");
        }, 1400);
      }
    });
  }

  function initChoiceFeedback() {
    document.addEventListener("change", function (event) {
      var target = event.target;
      if (!target || !target.matches) {
        return;
      }

      if (target.matches(".cart-quantity-form input, .add-to-cart-controls input")) {
        target.classList.add("is-updated");
        window.setTimeout(function () {
          target.classList.remove("is-updated");
        }, 520);
      }

      if (target.matches(".shipping-option input")) {
        var option = target.closest(".shipping-option");
        if (option) {
          option.classList.add("is-selected-now");
          window.setTimeout(function () {
            option.classList.remove("is-selected-now");
          }, 520);
        }
      }
    });

    document.addEventListener("click", function (event) {
      var swatch = event.target.closest && event.target.closest(".commercial-swatch, .variant-option, .category-pill");
      if (!swatch || reduceMotion) {
        return;
      }

      swatch.classList.add("is-pressed");
      window.setTimeout(function () {
        swatch.classList.remove("is-pressed");
      }, 220);
    });
  }

  function initHeroParallax() {
    if (reduceMotion || !window.matchMedia || !window.matchMedia("(pointer: fine) and (min-width: 900px)").matches) {
      return;
    }

    var stage = document.querySelector(".hero-brand-stage");
    if (!stage) {
      return;
    }

    var raf = 0;
    stage.addEventListener("pointermove", function (event) {
      if (raf) {
        window.cancelAnimationFrame(raf);
      }

      raf = window.requestAnimationFrame(function () {
        var rect = stage.getBoundingClientRect();
        var x = ((event.clientX - rect.left) / rect.width - 0.5).toFixed(3);
        var y = ((event.clientY - rect.top) / rect.height - 0.5).toFixed(3);
        stage.style.setProperty("--hero-parallax-x", x);
        stage.style.setProperty("--hero-parallax-y", y);
      });
    });

    stage.addEventListener("pointerleave", function () {
      stage.style.removeProperty("--hero-parallax-x");
      stage.style.removeProperty("--hero-parallax-y");
    });
  }

  function initReveal() {
    if (reduceMotion || !("IntersectionObserver" in window)) {
      return;
    }

    var selectors = [
      ".hero-copy",
      ".hero-brand-stage",
      ".section-heading",
      ".dna-grid",
      ".process-flow",
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
      ".checkout-stepper",
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

    var viewportHeight = window.innerHeight || document.documentElement.clientHeight;
    elements.forEach(function (element, index) {
      element.classList.add("motion-reveal");
      element.style.setProperty("--reveal-delay", Math.min(index % 4, 3) * 55 + "ms");
      var rect = element.getBoundingClientRect();
      if (rect.top < viewportHeight * 1.1 && rect.bottom > 0) {
        element.classList.add("motion-visible");
      }
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
    initActiveNavigation();
    initActionFeedback();
    initChoiceFeedback();
    initHeroParallax();
    initReveal();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
    return;
  }

  init();
})();
