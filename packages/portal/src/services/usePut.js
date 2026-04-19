export const usePut = async (url, body = {}, headers = {}, signal = null) => {

  const response = await fetch(url, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
         "X-Application-Type": "portal",
        ...headers,
      },
      body: JSON.stringify(body),
      signal: signal,
    });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Request failed with status ${response.status}: ${errorText}`);
  }

  return { status: response.status, response: await response.json() }

}
