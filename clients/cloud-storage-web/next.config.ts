

 import type {NextConfig} from "next";
 
 const nextConfig: NextConfig = {
  
  env: {
    SECRET_KEY: process.env.SECRET_KEY,
  },
   
  
  eslint: {
    ignoreDuringBuilds: true,
  },
 
   async rewrites() {
     return [
       
 
      
     ];
   },
 };
 
 export default nextConfig;