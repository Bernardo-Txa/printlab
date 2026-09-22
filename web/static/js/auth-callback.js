(function () {
  const status = document.querySelector('[data-auth-callback-status="true"]');
  const query = new URLSearchParams(window.location.search);
  const params = new URLSearchParams(window.location.hash.replace(/^#/, ''));
  const accessToken = params.get('access_token');
  const refreshToken = params.get('refresh_token');
  const expiresIn = Number.parseInt(params.get('expires_in') || '3600', 10);
  const type = params.get('type') || query.get('type') || '';
  const next = type === 'recovery' ? '/recuperar-senha/nova' : '/conta';

  function fail(message) {
    if (status) {
      status.textContent = message || 'Não foi possível confirmar seu acesso. Solicite um novo link e tente novamente.';
    }
  }

  if (query.get('error') || query.get('error_code')) {
    fail('Este link expirou ou já foi utilizado. Solicite um novo link.');
    return;
  }

  if (!accessToken || !refreshToken) {
    fail('Este link expirou ou já foi utilizado. Solicite um novo link.');
    return;
  }

  fetch('/auth/session', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      access_token: accessToken,
      refresh_token: refreshToken,
      expires_in: Number.isFinite(expiresIn) ? expiresIn : 3600,
      next: next,
    }),
  }).then(function (response) {
    if (!response.ok) {
      fail();
      return;
    }
    window.location.replace(next);
  }).catch(function () {
    fail();
  });
})();
