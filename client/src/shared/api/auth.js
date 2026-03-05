async function readError(response, fallbackMessage) {
  const errorMessage = await response.text();

  return errorMessage || fallbackMessage;
}

export async function loginRequest(credentials) {
  const response = await fetch("/api/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(credentials),
  });

  if (!response.ok) {
    throw new Error(await readError(response, "Ошибка аутентификации."));
  }

  return response.json();
}

export async function fetchProfile(accessToken) {
  const response = await fetch("/api/profile", {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(await readError(response, "Не удалось проверить сессию."));
  }

  return response.json();
}

export async function refreshSessionRequest(refreshToken) {
  const response = await fetch("/api/auth/refresh", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      refresh_token: refreshToken,
    }),
  });

  if (!response.ok) {
    throw new Error(await readError(response, "Не удалось обновить токен."));
  }

  return response.json();
}
