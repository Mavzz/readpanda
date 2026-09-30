export const useDelete = async (url, headers = {}, signal = null, body = null) => {

  const response = await fetch(url, {
      method: "DELETE",
      headers: {
        "Accept": "application/json",
        "X-Application-Type": "portal",
        ...(body ? { "Content-Type": "application/json" } : {}),
        ...headers,
      },
      ...(body ? { body: JSON.stringify(body) } : {}),
      signal: signal,
    });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Request failed with status ${response.status}: ${errorText}`);
  }

  // 204 No Content has no body
  if (response.status === 204) {
    return { status: 204, response: null };
  }

  return { status: response.status, response: await response.json() }

}
