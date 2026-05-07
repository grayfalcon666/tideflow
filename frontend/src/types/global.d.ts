export {}

declare global {
  // Asset module declarations
  declare module '*.svg'
  declare module '*.png'
  declare module '*.jpg'
  declare module '*.jpeg'
  declare module '*.gif'
  declare module '*.sass' {
    const content: string
    export default content
  }
  declare module '*.css' {
    const content: string
    export default content
  }

  interface Window {
    __tideflow_user_id__?: number
  }
}
