(function () {
  "use strict";

  function digitsOnly(value) {
    return String(value || "").replace(/\D/g, "");
  }

  function formatCPF(value) {
    var digits = digitsOnly(value).slice(0, 11);
    if (digits.length <= 3) {
      return digits;
    }
    if (digits.length <= 6) {
      return digits.slice(0, 3) + "." + digits.slice(3);
    }
    if (digits.length <= 9) {
      return digits.slice(0, 3) + "." + digits.slice(3, 6) + "." + digits.slice(6);
    }

    return digits.slice(0, 3) + "." + digits.slice(3, 6) + "." + digits.slice(6, 9) + "-" + digits.slice(9);
  }

  function formatPhone(value) {
    var digits = digitsOnly(value);
    if (digits.indexOf("55") === 0 && digits.length > 11) {
      digits = digits.slice(2);
    }
    digits = digits.slice(0, 11);

    if (digits.length <= 2) {
      return digits;
    }
    if (digits.length <= 6) {
      return "(" + digits.slice(0, 2) + ") " + digits.slice(2);
    }
    if (digits.length <= 10) {
      return "(" + digits.slice(0, 2) + ") " + digits.slice(2, 6) + "-" + digits.slice(6);
    }

    return "(" + digits.slice(0, 2) + ") " + digits.slice(2, 7) + "-" + digits.slice(7);
  }

  function formatCEP(value) {
    var digits = digitsOnly(value).slice(0, 8);
    if (digits.length <= 5) {
      return digits;
    }

    return digits.slice(0, 5) + "-" + digits.slice(5);
  }

  function setStatus(element, message, state) {
    if (!element) {
      return;
    }

    element.textContent = message || "";
    if (state) {
      element.dataset.state = state;
      return;
    }

    delete element.dataset.state;
  }

  function field(form, name) {
    var input = form.elements.namedItem(name);
    if (!input || typeof input.value !== "string") {
      return null;
    }

    return input;
  }

  function applyMask(input, formatter) {
    if (!input) {
      return;
    }

    input.value = formatter(input.value);
    input.addEventListener("input", function () {
      input.value = formatter(input.value);
    });
    input.addEventListener("blur", function () {
      input.value = formatter(input.value);
    });
  }

  function endpointForCEP(form, cep) {
    var base = form.dataset.cepLookupEndpoint || "/api/cep";
    return base.replace(/\/$/, "") + "/" + cep;
  }

  function fillAddress(form, address) {
    var mapping = {
      street: address.street,
      district: address.district,
      city: address.city,
      state: address.state,
    };

    Object.keys(mapping).forEach(function (name) {
      var input = field(form, name);
      if (input) {
        input.value = mapping[name] || "";
      }
    });
  }

  function init() {
    var form = document.querySelector("[data-checkout-form]");
    if (!form) {
      return;
    }

    var cpfInput = field(form, "cpf");
    var phoneInput = field(form, "phone");
    var cepInput = field(form, "postal_code");
    var status = form.querySelector("[data-cep-status]");

    applyMask(cpfInput, formatCPF);
    applyMask(phoneInput, formatPhone);
    applyMask(cepInput, formatCEP);

    if (!cepInput || !window.fetch || !window.AbortController) {
      return;
    }

    var timer = 0;
    var activeController = null;
    var lastLookup = "";

    function lookupCEP() {
      var cep = digitsOnly(cepInput.value).slice(0, 8);
      window.clearTimeout(timer);

      if (cep.length !== 8) {
        lastLookup = "";
        setStatus(status, "", "");
        return;
      }
      if (cep === lastLookup) {
        return;
      }

      lastLookup = cep;
      if (activeController) {
        activeController.abort();
      }
      activeController = new AbortController();
      setStatus(status, "Buscando CEP...", "loading");

      window.fetch(endpointForCEP(form, cep), {
        headers: { Accept: "application/json" },
        signal: activeController.signal,
      }).then(function (response) {
        if (response.status === 404) {
          throw new Error("not_found");
        }
        if (!response.ok) {
          throw new Error("unavailable");
        }

        return response.json();
      }).then(function (address) {
        fillAddress(form, address || {});
        setStatus(status, "Endereco encontrado. Revise os campos antes de continuar.", "success");
      }).catch(function (error) {
        if (error && error.name === "AbortError") {
          return;
        }

        lastLookup = "";
        if (error && error.message === "not_found") {
          setStatus(status, "CEP nao encontrado. Confira ou preencha o endereco manualmente.", "error");
          return;
        }

        setStatus(status, "Nao foi possivel consultar o CEP agora. Preencha o endereco manualmente.", "error");
      }).finally(function () {
        activeController = null;
      });
    }

    cepInput.addEventListener("input", function () {
      lastLookup = "";
      setStatus(status, "", "");
      window.clearTimeout(timer);
      if (digitsOnly(cepInput.value).length === 8) {
        timer = window.setTimeout(lookupCEP, 350);
      }
    });
    cepInput.addEventListener("blur", lookupCEP);
    cepInput.addEventListener("change", lookupCEP);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
    return;
  }

  init();
}());
