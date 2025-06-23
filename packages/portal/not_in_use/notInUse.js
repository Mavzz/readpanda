const coverUploadUrl = await getBackendUrl("/upload");
            const coverResponse = await UseFileUpload(
                coverUploadUrl,
                coverFile,
                "file",
                headers
            );
            console.log("Cover image upload response:", coverResponse.response.link);
            setUploadMessage(`Cover uploaded to: ${coverResponse.response.link}`);

            // Upload Book Manuscript
            const manuscriptUploadUrl = await getBackendUrl("/upload");
            const manuscriptResponse = await UseFileUpload(
                manuscriptUploadUrl,
                manuscriptFile,
                "file",
                headers
            );
            console.log(
                "Manuscript upload response:",
                manuscriptResponse.response.link
            );
            setUploadMessage(
                (prev) =>
                    prev + `Manuscript uploaded to: ${manuscriptResponse.response.link}`
            );