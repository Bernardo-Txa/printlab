(function () {
  const status = document.querySelector('[data-auth-callback-status="true"]');
  const params = new URLSearchParams(window.location.hash.replace(/^#/, ''));
  const accessToken = params.get('access_token');
  const refreshToken = params.get('refresh_token');
  const expiresIn = Number.parseInt(params.get('expires_in') || '3600', 10);
  const type = params.get('type') || new URLSearchParams(window.location.search).get('type') || '';
  const next = type === 'recovery' ? '/recuperar-senha/nova' : '/conta';

  function fail() {
    if (status) {
      status.textContent = 'Não foi possível confirmar o acesso. Solicite um novo link e tente novamente.';
    }
  }

  if (!accessToken || !refreshToken) {
    fail();
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
  }).catch(fail);
})();
