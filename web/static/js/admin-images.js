(function () {
  function find(form, selector) {
    return form.querySelector(selector);
  }

  function setStatus(form, message, isError) {
    var status = find(form, "[data-admin-image-status]");
    if (!status) {
      return;
    }
    status.textContent = message || "";
    status.classList.toggle("admin-image-status-error", Boolean(isError));
  }

  function progress(form, value, hidden) {
    var bar = find(form, "[data-admin-image-progress]");
    if (!bar) {
      return;
    }
    bar.hidden = Boolean(hidden);
    bar.value = value || 0;
  }

  function checked(form, name) {
    var field = form.elements[name];
    return Boolean(field && field.checked);
  }

  function value(form, name) {
    var field = form.elements[name];
    return field ? field.value : "";
  }

  function postJSON(url, body) {
    return fetch(url, {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      body: JSON.stringify(body)
    }).then(function (response) {
      return response.json().catch(function () {
        return {};
      }).then(function (payload) {
        if (!response.ok) {
          var error = new Error(payload.error || "request_failed");
          error.status = response.status;
          throw error;
        }
        return payload;
      });
    });
  }

  function uploadFile(uploadURL, file, form) {
    return new Promise(function (resolve, reject) {
      var request = new XMLHttpRequest();
      request.open("PUT", uploadURL);
      request.setRequestHeader("Content-Type", file.type);
      request.setRequestHeader("x-upsert", "false");
      request.upload.addEventListener("progress", function (event) {
        if (event.lengthComputable) {
          progress(form, Math.round((event.loaded / event.total) * 100), false);
        }
      });
      request.addEventListener("load", function () {
        if (request.status >= 200 && request.status < 300) {
          progress(form, 100, false);
          resolve();
          return;
        }
        reject(new Error("upload_failed"));
      });
      request.addEventListener("error", function () {
        reject(new Error("upload_failed"));
      });
      request.send(file);
    });
  }

  function submit(event) {
    var form = event.currentTarget;
    var fileInput = form.elements.image_file;
    var file = fileInput && fileInput.files ? fileInput.files[0] : null;
    event.preventDefault();
    setStatus(form, "", false);
    progress(form, 0, true);

    if (!file) {
      setStatus(form, "Selecione uma imagem.", true);
      return;
    }
    var maxSize = Number(form.dataset.maxSize || "0");
    if (maxSize > 0 && file.size > maxSize) {
      setStatus(form, "Arquivo acima do limite permitido.", true);
      return;
    }
    if (!/^(image\/jpeg|image\/png|image\/webp)$/.test(file.type)) {
      setStatus(form, "Formato de imagem nao permitido.", true);
      return;
    }

    var metadata = {
      variant_id: value(form, "variant_id"),
      content_type: file.type,
      file_size: file.size,
      filename: file.name
    };

    setStatus(form, "Autorizando upload...", false);
    postJSON(form.dataset.uploadUrl, metadata)
      .then(function (authorization) {
        setStatus(form, "Enviando imagem...", false);
        progress(form, 0, false);
        return uploadFile(authorization.upload_url, file, form).then(function () {
          return authorization;
        });
      })
      .then(function (authorization) {
        setStatus(form, "Finalizando cadastro...", false);
        return postJSON(form.dataset.finalizeUrl, {
          variant_id: value(form, "variant_id"),
          object_path: authorization.object_path,
          content_type: file.type,
          file_size: file.size,
          alt_text: value(form, "alt_text"),
          sort_order: Number(value(form, "sort_order") || "0"),
          is_primary: checked(form, "is_primary"),
          filename: file.name
        });
      })
      .then(function (result) {
        window.location.assign(result.redirect_url || window.location.pathname + "?ok=imagem");
      })
      .catch(function () {
        progress(form, 0, true);
        setStatus(form, "Nao foi possivel salvar a imagem.", true);
      });
  }

  document.querySelectorAll("[data-admin-image-upload]").forEach(function (form) {
    form.addEventListener("submit", submit);
  });
})();
