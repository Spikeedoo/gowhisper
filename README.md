# gowhisper
A lightweight API wrapper for whisper.cpp

### Endpoints
* POST /job
  * Body
    * ```{ audioUrl: <audio source URL> }```
  * Response
    * ```{ id: <unique job ID> }```
* GET /job/{jobId}
  * Response
    * ```{ id: <unique job ID>, audioUrl: <audio source URL>, transcript: <text transcript> }```