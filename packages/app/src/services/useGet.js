
export const useGet = async (url, headers = {}) => {

  const response =  await fetch(url, {
      method: "GET",
      headers: {
        "Accept": "application/json",
        ...headers,
      },
    });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Request failed with status ${response.status}: ${errorText}`);
  }

  return { status: response.status, response: await response.json()}

}

