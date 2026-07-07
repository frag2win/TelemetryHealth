output "instance_public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_instance.app_server.public_ip
}

output "dashboard_url" {
  description = "URL to access the React Dashboard"
  value       = "http://${aws_instance.app_server.public_ip}:5173"
}

output "api_url" {
  description = "URL to access the API"
  value       = "http://${aws_instance.app_server.public_ip}:8080"
}

output "ssh_command" {
  description = "Command to SSH into the instance"
  value       = "ssh -i telemetry-health-key.pem ubuntu@${aws_instance.app_server.public_ip}"
}
