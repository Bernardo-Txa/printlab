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
      request.setRequestHeader("cache-control", "public, max-age=31536000, immutable");
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

  function optimizedImage(file) {
    if (!window.createImageBitmap || !window.OffscreenCanvas && !document.createElement) {
      return Promise.resolve(file);
    }
    return window.createImageBitmap(file, { imageOrientation: "from-image" }).then(function (bitmap) {
      var maxWidth = 1200;
      var scale = Math.min(1, maxWidth / bitmap.width);
      var width = Math.max(1, Math.round(bitmap.width * scale));
      var height = Math.max(1, Math.round(bitmap.height * scale));
      var canvas = document.createElement("canvas");
      canvas.width = width;
      canvas.height = height;
      canvas.getContext("2d").drawImage(bitmap, 0, 0, width, height);
      bitmap.close();
      return new Promise(function (resolve) {
        canvas.toBlob(function (blob) {
          if (!blob) {
            resolve(file);
            return;
          }
          resolve(new File([blob], file.name.replace(/\.[^.]+$/, "") + ".webp", { type: "image/webp" }));
        }, "image/webp", 0.82);
      });
    }).catch(function () {
      return file;
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
    if (!/^(image\/jpeg|image\/png|image\/webp)$/.test(file.type)) {
      setStatus(form, "Formato de imagem nao permitido.", true);
      return;
    }

    var maxSize = Number(form.dataset.maxSize || "0");
    setStatus(form, "Otimizando imagem...", false);
    optimizedImage(file).then(function (optimized) {
      if (maxSize > 0 && optimized.size > maxSize) {
        throw new Error("image_too_large");
      }
      return optimized;
    }).then(function (optimized) {
      var metadata = { variant_id: value(form, "variant_id"), content_type: optimized.type, file_size: optimized.size, filename: optimized.name };
      setStatus(form, "Autorizando upload...", false);
      return postJSON(form.dataset.uploadUrl, metadata).then(function (authorization) {
        return uploadFile(authorization.upload_url, optimized, form).then(function () { return { authorization: authorization, file: optimized }; });
      });
    })
      .then(function (result) {
        var authorization = result.authorization;
        var uploadedFile = result.file;
        setStatus(form, "Finalizando cadastro...", false);
        return postJSON(form.dataset.finalizeUrl, {
          variant_id: value(form, "variant_id"),
          object_path: authorization.object_path,
          content_type: uploadedFile.type,
          file_size: uploadedFile.size,
          alt_text: value(form, "alt_text"),
          sort_order: Number(value(form, "sort_order") || "0"),
          is_primary: checked(form, "is_primary"),
          filename: uploadedFile.name
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
