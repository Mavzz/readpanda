export const useDelete = async (url, headers = {}, signal = null) => {

  const response = await fetch(url, {
      method: "DELETE",
      headers: {
        "Accept": "application/json",
        "X-Application-Type": "portal",
        ...headers,
      },
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
