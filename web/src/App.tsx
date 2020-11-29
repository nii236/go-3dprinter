import * as React from "react"
import MoreVertIcon from "@material-ui/icons/MoreVert"
import FavoriteIcon from "@material-ui/icons/Favorite"
import ShareIcon from "@material-ui/icons/Share"
import { Alert, Skeleton } from "@material-ui/lab"
import "./App.css"
import Container from "@material-ui/core/Container"
import Typography from "@material-ui/core/Typography"
import Box from "@material-ui/core/Box"
import Link from "@material-ui/core/Link"
import { DropzoneArea } from "material-ui-dropzone"
import { Button, Card, CardActions, CardContent, CardHeader, Divider, IconButton, List, ListItem, Menu, MenuItem, Paper } from "@material-ui/core"
import { Action, useMutation, useQuery } from "react-fetching-library"
import { createClient, ClientContextProvider } from "react-fetching-library"
import { Pause, Printer, StopCircle } from "react-feather"
import { Refresh } from "@material-ui/icons"
import { useInterval } from "react-use"

const client = createClient({})

interface ControlPanelProps {
	sessionId?: string
	gcode?: string
	status?: string
}

function App() {
	const [sessionId, setSession] = React.useState<string | undefined>()
	return (
		<ClientContextProvider client={client}>
			<Box margin={4}>
				<Container maxWidth="sm">
					<Box display={"flex"} flexDirection={"column"}>
						<Box margin={2}>
							<ControlPanel sessionId={sessionId} />
						</Box>
						<Box margin={2}>
							<Sessions setSession={setSession} />
						</Box>
						<Box margin={2}>
							<Files sessionId={sessionId} />
						</Box>
					</Box>
				</Container>
			</Box>
		</ClientContextProvider>
	)
}

interface PrinterInfo {
	busy: boolean
	status: string
}
const ControlPanel = (props: ControlPanelProps) => {
	const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null)
	const [err, setErr] = React.useState<string | null>()
	const { mutate: startPrint } = useMutation<APIResponse<string>, {}, { sessionId: string }>(startPrintRequester)
	const { mutate: pausePrint } = useMutation<APIResponse<string>, {}, { sessionId: string }>(pausePrintRequester)
	const { mutate: cancelPrint } = useMutation<APIResponse<string>, {}, { sessionId: string }>(cancelPrintRequester)
	const { mutate: autoHome } = useMutation<APIResponse<string>, {}, { sessionId: string }>(autoHomeRequester)
	const { mutate: printLevelTest } = useMutation<APIResponse<string>, {}, { sessionId: string }>(printLevelTestRequester)
	const { loading, payload: printerInfoResponse, query: queryPrinterInfo, error, errorObject } = useQuery<APIResponse<PrinterInfo>>(
		{ endpoint: "/api/printer/info", method: "GET", responseType: "json" },
		true
	)
	const [printerInfo, setPrinterInfo] = React.useState<PrinterInfo>()
	setInterval(() => {
		queryPrinterInfo()
		setPrinterInfo(printerInfoResponse?.payload)
	}, 5000)

	if (loading) {
		return <Skeleton variant="text" />
	}
	// if (error) {
	// 	setErr(errorObject)
	// }
	const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
		setAnchorEl(event.currentTarget)
	}
	if (err && props.sessionId) setErr(null)
	const busyText = printerInfo?.busy ? `Printer is currently busy` : `Printer is currently not busy`
	return (
		<Card>
			<Menu keepMounted open={Boolean(anchorEl)} anchorEl={anchorEl} onClose={() => setAnchorEl(null)}>
				<MenuItem
					onClick={() => {
						if (!props.sessionId) {
							setErr("session not selected")
							setAnchorEl(null)
							return
						}
						autoHome({ sessionId: props.sessionId })
						setAnchorEl(null)
					}}
				>
					Auto Home
				</MenuItem>
				<MenuItem
					onClick={() => {
						if (!props.sessionId) {
							setErr("session not selected")
							setAnchorEl(null)
							return
						}
						printLevelTest({ sessionId: props.sessionId })
						setAnchorEl(null)
					}}
				>
					Print level test
				</MenuItem>
			</Menu>

			<CardHeader
				title={"Printer Status"}
				action={
					<div>
						<IconButton aria-label="settings" aria-controls="printer-menu" aria-haspopup="true" onClick={handleClick}>
							<MoreVertIcon />
						</IconButton>
					</div>
				}
			/>
			<CardContent>
				{err && <Alert severity="error">{err}</Alert>}
				<List>
					<ListItem>
						<Typography variant="body2" color="textSecondary" component="p">
							Selected session: {props.sessionId ? props.sessionId : "None"}
						</Typography>
					</ListItem>
					<ListItem>
						<Typography variant="body2" color="textSecondary" component="p">
							Selected gcode: {props.gcode ? props.gcode : "None"}
						</Typography>
					</ListItem>
					<ListItem>
						<Typography variant="body2" color="textSecondary" component="p">
							Status: {printerInfo?.payload.status}
						</Typography>
						<Typography variant="body2" color="textSecondary" component="p">
							{busyText}
						</Typography>
					</ListItem>
				</List>
			</CardContent>
			<CardActions disableSpacing>
				<IconButton
					aria-label="print"
					onClick={() => {
						if (!props.sessionId) {
							setErr("Session not selected")
							return
						}
						startPrint({ sessionId: props.sessionId })
					}}
				>
					<Printer />
				</IconButton>
				<IconButton
					aria-label="pause"
					onClick={() => {
						if (!props.sessionId) {
							setErr("Session not selected")
							return
						}
						pausePrint({ sessionId: props.sessionId })
					}}
				>
					<Pause />
				</IconButton>
				<IconButton
					aria-label="cancel"
					onClick={() => {
						if (!props.sessionId) {
							setErr("Session not selected")
							return
						}
						cancelPrint({ sessionId: props.sessionId })
					}}
				>
					<StopCircle />
				</IconButton>
			</CardActions>
		</Card>
	)
}
interface File {
	id: string
	name: string
}
interface APIResponse<T> {
	payload: T
}
interface LoadFileRequest {
	session_id: string
	file_id: string
}
const autoHomeRequester = (payload: { sessionId: string }): Action => {
	return { endpoint: "/api/command/autohome", method: "POST", responseType: "json", body: payload }
}
const printLevelTestRequester = (payload: { sessionId: string }): Action => {
	return { endpoint: "/api/command/levelbedtest", method: "POST", responseType: "json", body: payload }
}
const cancelPrintRequester = (payload: { sessionId: string }): Action => {
	return { endpoint: "/api/command/cancel", method: "POST", responseType: "json", body: payload }
}
const startPrintRequester = (payload: { sessionId: string }): Action => {
	return { endpoint: "/api/command/start", method: "POST", responseType: "json", body: payload }
}
const pausePrintRequester = (payload: { sessionId: string }): Action => {
	return { endpoint: "/api/command/pause", method: "POST", responseType: "json", body: payload }
}
const loadFileRequester = (payload: LoadFileRequest): Action => {
	return { endpoint: "/api/command/load", method: "POST", responseType: "json", body: payload }
}
interface SessionsProps {
	setSession: (sessionID: string) => void
}
const Sessions = (props: SessionsProps) => {
	const { loading, payload, error, query, errorObject } = useQuery<APIResponse<string[]>>(
		{ endpoint: "/api/printer/sessions", method: "GET", responseType: "json" },
		true
	)
	if (loading) {
		return <Skeleton variant="text" />
	}
	if (!payload || !payload.payload) {
		return <Alert severity="warning">No data</Alert>
	}
	return (
		<Card>
			<CardHeader
				title={"Available sessions"}
				action={
					<IconButton
						aria-label="refresh"
						onClick={() => {
							query()
						}}
					>
						<Refresh />
					</IconButton>
				}
			/>
			<CardContent>
				<Button
					onClick={async () => {
						try {
							await query()
						} catch (err) {
							console.error(err)
						}
					}}
				>
					Refresh
				</Button>
				{error && <Alert severity="error">{errorObject}</Alert>}

				<List>
					{payload.payload.map((sessionID) => {
						return (
							<div key={sessionID}>
								<ListItem
									button
									onClick={() => {
										props.setSession(sessionID)
									}}
								>
									{sessionID}
								</ListItem>
								<Divider />
							</div>
						)
					})}
				</List>
			</CardContent>
		</Card>
	)
}
interface FilesProps {
	sessionId?: string
}
const Files = (props: FilesProps) => {
	const { loading, payload, error, query, errorObject } = useQuery<APIResponse<File[]>>({ endpoint: "/api/gcodes", method: "GET", responseType: "json" }, true)
	const { mutate: loadFile } = useMutation<APIResponse<string>, {}, LoadFileRequest>(loadFileRequester)
	const [err, setErr] = React.useState<string | undefined>()
	if (props.sessionId && err) {
		setErr(undefined)
	}
	if (loading) {
		return <Skeleton variant="text" />
	}
	if (!payload || !payload.payload) {
		return <Alert severity="warning">No data</Alert>
	}
	return (
		<Card>
			<CardHeader
				title={"Available files"}
				action={
					<IconButton
						aria-label="refresh"
						onClick={() => {
							query()
						}}
					>
						<Refresh />
					</IconButton>
				}
			/>
			<CardContent>
				<Button
					onClick={async () => {
						try {
							await query()
						} catch (err) {
							console.error(err)
						}
					}}
				>
					Refresh
				</Button>
				{error && <Alert severity="error">{errorObject}</Alert>}
				{err && <Alert severity="error">{err}</Alert>}
				<List>
					{payload.payload.map((item) => {
						return (
							<div key={item.id}>
								<ListItem
									key={item.id}
									button
									onClick={() => {
										if (!props.sessionId) {
											setErr("session not selected")
											return
										}
										loadFile({ session_id: props.sessionId, file_id: item.id })
									}}
								>
									{item.name}
								</ListItem>
								<Divider />
							</div>
						)
					})}
				</List>
				<DropzoneArea onChange={() => {}} />
			</CardContent>
		</Card>
	)
}

export default App
