export const useGet = async (url, headers = {}, signal = null) => {

  const response =  await fetch(url, {
      method: "GET",
      headers: {
        "Accept": "application/json",
        ...headers,
      },
      signal: signal,
    });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Request failed with status ${response.status}: ${errorText}`);
  }

  return { status: response.status, response: await response.json()}

}
